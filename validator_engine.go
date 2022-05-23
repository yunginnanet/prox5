package prox5

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	//	"net/url"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"
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
		TLSHandshakeTimeout: s.swampopt.validationTimeout.Load().(time.Duration),
	}

	return client, transporter, req, err
}

func (sock *Proxy) bad() {
	sock.timesBad.Store(sock.timesBad.Load().(int) + 1)
}

func (sock *Proxy) good() {
	sock.timesValidated.Store(sock.timesValidated.Load().(int) + 1)
	sock.lastValidated.Store(time.Now())
}

func (s *Swamp) checkHTTP(sock *Proxy) (string, error) {
	var (
		client      *http.Client
		transporter *http.Transport
		req         *http.Request
		err         error
	)

	if client, transporter, req, err = s.prepHTTP(); err != nil {
		return "", err
	}

	var dialSocks = socks.Dial(fmt.Sprintf(
		"socks%s://%s/?timeout=%ss",
		sock.Proto.Load().(string),
		sock.Endpoint,
		s.GetValidationTimeoutStr()),
	)

	var transportDialer = dialSocks
	if sock.Proto.Load().(string) == "none" {
		transportDialer = proxy.Direct.Dial
	}

	// if sock.Proto.Load().(string) != "http" {
	transporter.Dial = transportDialer

	// } else {
	//	if purl, err := url.Parse("http://" + sock.Endpoint); err == nil {
	//		transporter.Proxy = http.ProxyURL(purl)
	//	} else {
	//		return "", err
	//	}
	// }

	client = &http.Client{
		Transport: transporter,
		Timeout:   s.swampopt.validationTimeout.Load().(time.Duration),
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
	s.Stats.Checked++
}

func (s *Swamp) singleProxyCheck(sock *Proxy) error {
	defer s.anothaOne()
	split := strings.Split(sock.Endpoint, "@")
	endpoint := split[0]
	if len(split) == 2 {
		endpoint = split[1]
	}
	if _, err := net.DialTimeout("tcp", endpoint,
		s.swampopt.validationTimeout.Load().(time.Duration)); err != nil {
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

func (sock *Proxy) validate() {
	var sversions = []string{"4", "5", "4a"}

	s := sock.parent
	if s.useProx.Check(sock) {
		// s.dbgPrint(ylw + "useProx ratelimited: " + sock.Endpoint + rst)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		return
	}

	// determined as bad, won't try again until it expires from that cache
	if s.badProx.Peek(sock) {
		s.dbgPrint(ylw + "badProx ratelimited: " + sock.Endpoint + rst)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		return
	}

	// try to use the proxy with all 3 SOCKS versions
	var good = false
	for _, sver := range sversions {
		if s.Status.Load().(SwampStatus) == Paused {
			return
		}

		sock.Proto.Store(sver)
		if err := s.singleProxyCheck(sock); err == nil {
			//			if sock.Proto != "http" {
			s.dbgPrint(grn + "verified " + sock.Endpoint + " as SOCKS" + sver + rst)
			//			} else {
			//				s.dbgPrint(ylw + "verified " + sock.Endpoint + " as http (not usable yet)" + rst)
			//			}
			good = true
			break
		}
	}

	if !good {
		s.dbgPrint(red + "failed to verify: " + sock.Endpoint + rst)
		sock.bad()
		s.badProx.Check(sock)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		return
	}

	sock.good()
	atomic.StoreUint32(&sock.lock, stateUnlocked)

	switch sock.Proto.Load().(string) {
	case "4":
		go func() {
			s.Stats.v4()
			s.ValidSocks4 <- sock
		}()
		return
	case "4a":
		go func() {
			s.Stats.v4a()
			s.ValidSocks4a <- sock
		}()
		return
	case "5":
		go func() {
			s.Stats.v5()
			s.ValidSocks5 <- sock
		}()
		return
	default:
		return
	}
}
