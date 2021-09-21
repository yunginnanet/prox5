package pxndscvm

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"
	"h12.io/socks"
)

var failcount = 0

func (s *Swamp) checkHTTP(sock *Proxy) (string, error) {
	req, err := http.NewRequest("GET", s.GetRandomEndpoint(), bytes.NewBuffer([]byte("")))
	if err != nil {
		return "", err
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

	var dialSocks = socks.Dial("socks" + sock.Proto + "://" +
		sock.Endpoint + "?timeout=" + strconv.Itoa(s.GetValidationTimeout()) + "s")

	var client *http.Client
	var transporter = dialSocks

	if sock.Proto == "none" {
		transporter = proxy.Direct.Dial
	}

	//goland:noinspection GoDeprecation
	client = &http.Client{
		Transport: &http.Transport{
			Dial:                transporter,
			DisableKeepAlives:   true,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: time.Duration(s.GetValidationTimeout()) * time.Second,
		},
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

	rbody, err := ioutil.ReadAll(resp.Body)
	return string(rbody), err
}

func (s *Swamp) singleProxyCheck(sock *Proxy) error {
	if _, err := net.DialTimeout("tcp", sock.Endpoint, time.Duration(s.GetValidationTimeout())*time.Second); err != nil {
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
	var sversions = []string{"4", "4a", "5"}

	s := sock.parent
	if s.useProx.Check(sock) {
		s.dbgPrint(ylw + "useProx ratelimited: " + sock.Endpoint + rst)
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
		if s.Status == Paused {
			return
		}
		sock.Proto = sver
		if err := s.singleProxyCheck(sock); err == nil {
			s.dbgPrint(grn + "verified " + sock.Endpoint + " as SOCKS" + sver + rst)
			good = true
			break
		}
	}

	if !good {
		if failcount > 100 {
			s.dbgPrint(ylw + "failed to verify ~100 proxies, last: " + sock.Endpoint)
			failcount = 0
		}
		sock.TimesBad++
		s.badProx.Check(sock)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		return
	}

	sock.TimesValidated++
	sock.LastVerified = time.Now()
	atomic.StoreUint32(&sock.lock, stateUnlocked)

	switch sock.Proto {
	case "4":
		go s.Stats.v4()
		s.ValidSocks4 <- sock
	case "4a":
		go s.Stats.v4a()
		s.ValidSocks4a <- sock
	case "5":
		go s.Stats.v5()
		s.ValidSocks5 <- sock
	}
}
