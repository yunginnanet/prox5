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
	"time"

	"golang.org/x/net/proxy"
	"h12.io/socks"
)

func (s *Swamp) feed() {
	s.dbgPrint("swamp feed start")
	for {
		if s.Status == Paused {
			return
		}
		select {
		case s.Pending <- randStrChoice(s.scvm):
			//
		case <-s.quit:
			s.dbgPrint("feed() paused")
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Swamp) checkHTTP(sock Proxy) (string, error) {
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

func (s *Swamp) singleProxyCheck(sock Proxy) error {
	if _, err := net.DialTimeout("tcp", sock.Endpoint, time.Duration(s.GetValidationTimeout())*time.Second); err != nil {
		badProx.Check(sock)
		return err
	}

	resp, err := s.checkHTTP(sock)
	if err != nil {
		badProx.Check(sock)
		return err
	}

	if newip := net.ParseIP(resp); newip == nil {
		badProx.Check(sock)
		return errors.New("bad response from http request: " + resp)
	}

	sock.ProxiedIP = resp
	return nil
}

func (s *Swamp) validate() {
	var sversions = []string{"5", "4", "4a"}
	for {

		if s.Status == Paused {
			return
		}

		sock := <-s.Pending
		p := Proxy{
			Endpoint: sock,
		}

		// ratelimited
		if useProx.Check(p) {
			s.dbgPrint(ylw+"useProx ratelimited: " + p.Endpoint+rst)
			continue
		}

		// determined as bad, won't try again until it expires from that cache
		if badProx.Peek(p) {
			s.dbgPrint(ylw + "badProx ratelimited: " + p.Endpoint + rst)
			continue
		}

		// try to use the proxy with all 3 SOCKS versions
		var good = false
		for _, sver := range sversions {
			if s.Status == Paused {
				return
			}
			p.Proto = sver
			if err := s.singleProxyCheck(p); err == nil {
				s.dbgPrint(grn + "verified " + p.Endpoint + " as SOCKS" + sver + rst)
				good = true
				break
			}
		}

		if !good {
			// s.dbgPrint(ylw + "failed to verify " + p.Endpoint + rst)
			badProx.Check(p)
			continue
		}

		p.Verified = time.Now()

		switch p.Proto {
		case "4":
			go s.Stats.v4()
			s.Socks4 <- p
		case "4a":
			go s.Stats.v4a()
			s.Socks4a <- p
		case "5":
			go s.Stats.v5()
			s.Socks5 <- p
		}
	}
}

func (s *Swamp) tossUp() {
	s.dbgPrint("tossUp() proxy checking loop start")
	for {
		if s.Status == Paused {
			return
		}
		select {
		case <-s.quit:
			s.dbgPrint("tossUp() paused")
			return
		default:
			go s.pool.Submit(s.validate)
			time.Sleep(time.Duration(10) * time.Millisecond)
		}
	}
}
