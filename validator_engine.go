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

func (pe *ProxyEngine) prepHTTP() (*http.Client, *http.Transport, *http.Request, error) {
	req, err := http.NewRequest("GET", pe.GetRandomEndpoint(), bytes.NewBuffer([]byte("")))
	if err != nil {
		return nil, nil, nil, err
	}
	headers := make(map[string]string)
	headers["User-Agent"] = pe.RandomUserAgent()
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
	headers["Accept-Language"] = "en-US,en;q=0.5"
	headers["'Accept-Encoding'"] = "gzip, deflate, br"
	headers["Connection"] = "keep-alive"
	for header, value := range headers {
		req.Header.Set(header, value)
	}
	var client = &http.Client{}
	var transporter = &http.Transport{
		DisableKeepAlives:   true,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout: pe.swampopt.validationTimeout,
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

func (pe *ProxyEngine) bakeHTTP(sock *Proxy) (client *http.Client, req *http.Request, err error) {
	dialSocks := socks.Dial(fmt.Sprintf(
		"socks%s://%s/?timeout=%ss",
		getProtoStr(sock.proto),
		sock.Endpoint,
		pe.GetValidationTimeoutStr()),
	)
	var (
		purl      *url.URL
		transport *http.Transport
	)

	if client, transport, req, err = pe.prepHTTP(); err != nil {
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

func (pe *ProxyEngine) checkHTTP(sock *Proxy) (string, error) {
	var (
		client *http.Client
		req    *http.Request
		err    error
	)

	client, req, err = pe.bakeHTTP(sock)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	rbody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return string(rbody), err
}

func (pe *ProxyEngine) anothaOne() {
	pe.stats.Checked++
}

func (pe *ProxyEngine) singleProxyCheck(sock *Proxy) error {
	defer pe.anothaOne()
	split := strings.Split(sock.Endpoint, "@")
	endpoint := split[0]
	if len(split) == 2 {
		endpoint = split[1]
	}
	if _, err := net.DialTimeout("tcp", endpoint,
		pe.swampopt.validationTimeout); err != nil {
		pe.badProx.Check(sock)
		return err
	}

	resp, err := pe.checkHTTP(sock)
	if err != nil {
		pe.badProx.Check(sock)
		return err
	}

	if newip := net.ParseIP(resp); newip == nil {
		pe.badProx.Check(sock)
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
	if str, ok := protoMap[protocol]; ok {
		return str
	}
	panic(protoMap[protocol])
}

func (sock *Proxy) validate() {
	atomic.StoreUint32(&sock.lock, stateLocked)
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	s := sock.parent
	if s.useProx.Check(sock) {
		// s.dbgPrint("useProx ratelimited: " + sock.Endpoint )
		return
	}

	// determined as bad, won't try again until it expires from that cache
	if s.badProx.Peek(sock) {
		s.dbgPrint("badProx ratelimited: " + sock.Endpoint)
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
		s.dbgPrint("verified " + sock.Endpoint + " as SOCKS" + getProtoStr(sock.proto))
		break
	default:
		s.dbgPrint("failed to verify: " + sock.Endpoint)
		sock.bad()
		s.badProx.Check(sock)
		return
	}

	sock.good()
	s.tally(sock)
}

func (pe *ProxyEngine) tally(sock *Proxy) {
	switch sock.proto {
	case ProtoSOCKS4:
		pe.stats.v4()
		pe.Valids.SOCKS4 <- sock
	case ProtoSOCKS4a:
		pe.stats.v4a()
		pe.Valids.SOCKS4a <- sock
	case ProtoSOCKS5:
		pe.stats.v5()
		pe.Valids.SOCKS5 <- sock
	case ProtoHTTP:
		pe.stats.http()
		pe.Valids.HTTP <- sock
	default:
		return
	}
}
