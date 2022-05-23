package prox5

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"h12.io/socks"
)

func (s *Swamp) prepHTTP() (*http.Client, *http.Transport, *http.Request, error) {
	req, err := http.NewRequest("GET", s.GetRandomEndpoint(), bytes.NewBuffer([]byte("")))
	if err != nil {
		return nil, nil, nil, err
	}
	headers := make(map[string]string)
	headers["User-Agent"] = s.RandomUserAgent()
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
	headers["Accept-Language"] = "en-US,en;q=0.5"
	headers["'Accept-Encoding'"] = "gzip, deflate, br"
	headers["Connection"] = "keep-alive"
	for header, value := range headers {
		req.Header.Set(header, value)
	}
	var client *http.Client
	var transporter = &http.Transport{
		DisableKeepAlives:   true,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: s.swampopt.validationTimeout,
	}

	return client, transporter, req, err
}

func (sock *Proxy) bad() {
	atomic.AddInt64(&sock.timesBad, 1)
}

func (sock *Proxy) good() {
	atomic.AddInt64(&sock.timesValidated, 1)
	sock.lastValidated = time.Now()
}

func (s *Swamp) bakeHTTP(sock *Proxy) (client *http.Client, req *http.Request, err error) {
	dialSocks := socks.Dial(fmt.Sprintf(
		"socks%s://%s/?timeout=%ss",
		getProtoStr(sock.proto),
		sock.Endpoint,
		s.GetValidationTimeoutStr()),
	)
	var (
		purl      *url.URL
		transport *http.Transport
	)

	if client, transport, req, err = s.prepHTTP(); err != nil {
		return
	}

	if sock.proto != ProtoHTTP {
		transport.Dial = dialSocks
		client.Transport = transport
		return
	}
	if purl, err = url.Parse("http://" + sock.Endpoint); err != nil {
		return
	}
	transport.Proxy = http.ProxyURL(purl)
	return
}

func (s *Swamp) checkHTTP(sock *Proxy) (string, error) {
	var (
		client *http.Client
		req    *http.Request
		err    error
	)

	client, req, err = s.bakeHTTP(sock)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	rbody, err := io.ReadAll(resp.Body)
	return string(rbody), err
}

func (s *Swamp) anothaOne() {
	s.stats.Checked++
}

func (s *Swamp) singleProxyCheck(sock *Proxy) error {
	defer s.anothaOne()
	split := strings.Split(sock.Endpoint, "@")
	endpoint := split[0]
	if len(split) == 2 {
		endpoint = split[1]
	}
	if _, err := net.DialTimeout("tcp", endpoint,
		s.swampopt.validationTimeout); err != nil {
		s.badProx.Check(sock)
		return err
	}

	resp, err := s.checkHTTP(sock)
	if err != nil {
		s.badProx.Check(sock)
		return err
	}

	if newip := net.ParseIP(resp); newip == nil {
		s.badProx.Check(sock)
		return errors.New("bad response from http request: " + resp)
	}

	sock.ProxiedIP = resp

	return nil
}

var protoMap = map[ProxyProtocol]string{
	ProtoSOCKS4: "4a", ProtoSOCKS4a: "4",
	ProtoSOCKS5: "5", ProtoHTTP: "http",
}

func getProtoStr(protocol ProxyProtocol) string {
	return protoMap[protocol]
}

func (sock *Proxy) validate() {
	atomic.StoreUint32(&sock.lock, stateLocked)
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	s := sock.parent
	if s.useProx.Check(sock) {
		// s.dbgPrint(ylw + "useProx ratelimited: " + sock.Endpoint + rst)
		return
	}

	// determined as bad, won't try again until it expires from that cache
	if s.badProx.Peek(sock) {
		s.dbgPrint(ylw + "badProx ratelimited: " + sock.Endpoint + rst)
		return
	}

	// TODO: consider giving the option for verbose logging of this stuff?

	// try to use the proxy with all 3 SOCKS versions
	for proto := range protoMap {
		select {
		case <-s.ctx.Done():
			return
		default:
			sock.proto = proto
			if err := s.singleProxyCheck(sock); err != nil {
				// if the proxy is no good, we continue on to the next.
				continue
			}
			break
		}
	}

	switch sock.proto {
	case ProtoSOCKS4, ProtoSOCKS4a, ProtoSOCKS5, ProtoHTTP:
		s.dbgPrint(grn + "verified " + sock.Endpoint + " as SOCKS" + getProtoStr(sock.proto) + rst)
		break
	default:
		s.dbgPrint(red + "failed to verify: " + sock.Endpoint + rst)
		sock.bad()
		s.badProx.Check(sock)
		return
	}

	sock.good()
	s.tally(sock)
}

func (s *Swamp) tally(sock *Proxy) {
	switch sock.proto {
	case ProtoSOCKS4:
		s.stats.v4()
		s.ValidSocks4 <- sock
	case ProtoSOCKS4a:
		s.stats.v4a()
		s.ValidSocks4a <- sock
	case ProtoSOCKS5:
		s.stats.v5()
		s.ValidSocks5 <- sock
	case ProtoHTTP:
		s.stats.http()
		s.ValidHTTP <- sock
	default:
		return
	}
}
