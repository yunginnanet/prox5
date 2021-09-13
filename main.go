package pxndscvm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/alitto/pond"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/proxy"
	"h12.io/socks"
)

const (
	grn = "\033[32m"
	red = "\033[31m"
	rst = "\033[0m"
)

// LoadProxyTXT loads proxies from a given seed file and randomly feeds them to the workers.
// This fucntion has no real error handling, if the file can't be opened it's gonna straight up panic.
// TODO: make it more gooder.
func (s *Swamp) LoadProxyTXT(seedFile string) error {
	s.dbgPrint("LoadProxyTXT start")

	f, err := os.Open(seedFile)
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(f)
	go s.tossUp()
	for scan.Scan() {
		s.scvm = append(s.scvm, scan.Text())
	}
	go s.feed()
	if err := f.Close(); err != nil {
		s.dbgPrint(err.Error())
		return err
	}
	return nil
}

func (s *Swamp) feed() {
	s.dbgPrint("swamp feed start")
	for {
		select {
		case s.Pending <- RandStrChoice(s.scvm):
			//
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// MysteryDial will return a dialer that will use a different proxy for every request.
func (s *Swamp) MysteryDial(ctx context.Context, network, addr string) (net.Conn, error) {
	var sock *Proxy
	sock = &Proxy{Endpoint: ""}
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
		candidate := s.getProxy()
		if !s.stillGood(candidate) {
			continue
		}

		sock = candidate

		if sock.Endpoint != "" {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var dialSocks = socks.Dial("socks" + sock.Proto + "://" + sock.Endpoint + "?timeout=3s")

	return dialSocks(network, addr)
}

func (s *Swamp) proxyGETRequest(sock *Proxy) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, err := http.NewRequest("GET", RandStrChoice(myipsites), bytes.NewBuffer([]byte("")))
	if err != nil {
		return "", err
	}

	headers := make(map[string]string)
	// headers["Host"] = "wtfismyip.com"
	headers["User-Agent"] = RandStrChoice(s.swampopt.UserAgents)
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
	headers["Accept-Language"] = "en-US,en;q=0.5"
	headers["'Accept-Encoding'"] = "gzip, deflate, br"
	headers["Connection"] = "keep-alive"

	for header, value := range headers {
		req.Header.Set(header, value)
	}

	var dialSocks = socks.Dial("socks" + sock.Proto + "://" + sock.Endpoint + "?timeout=4s")
	var client *http.Client

	if sock.Proto == "none" {
		//goland:noinspection GoDeprecation
		client = &http.Client{
			Transport: &http.Transport{
				Dial:                proxy.Direct.Dial,
				DisableKeepAlives:   true,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				TLSHandshakeTimeout: time.Duration(4) * time.Second,
			},
		}
	} else {
		//goland:noinspection GoDeprecation
		client = &http.Client{
			Transport: &http.Transport{
				Dial:                dialSocks,
				DisableKeepAlives:   true,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				TLSHandshakeTimeout: time.Duration(4) * time.Second,
			},
		}
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, err := net.DialTimeout("tcp", sock.Endpoint, 8*time.Second); err != nil {
		badProx.Check(sock)
		return err
	}
	resp, err := s.proxyGETRequest(sock)
	if err != nil {
		badProx.Check(sock)
		return err
	}
	if newip := net.ParseIP(resp); newip == nil {
		badProx.Check(sock)
		return errors.New("nil response from http request")
	}

	return nil
}

func (s *Swamp) tossUp() {
	s.dbgPrint("tossUp() proxy checking loop start")
	var sversions = []string{"5", "4", "4a"}
	s.Birthday = time.Now()
	panicHandler := func(p interface{}) {
		log.Error().Interface("panic", p).Msg("Task panicked")
	}
	pool := pond.New(100, 10000, pond.MinWorkers(100), pond.PanicHandler(panicHandler))
	for {
		pool.Submit(func() {
			for {
				sock := <-s.Pending
				p := &Proxy{
					Endpoint: sock,
				}
				// ratelimited
				if useProx.Check(p) {
					s.dbgPrint("useProx ratelimited: " + p.Endpoint)
					continue
				}
				// determined as bad, won't try again until it expires from that cache
				if badProx.Peek(p) {
					s.dbgPrint("badProx ratelimited: " + p.Endpoint)
					continue
				}

				// try to use the proxy with all 3 SOCKS versions
				var good = false
				for _, sver := range sversions {
					p.Proto = sver
					if err := s.singleProxyCheck(p); err == nil {
						s.dbgPrint(grn+"verified " + p.Endpoint + " as SOCKS" + sver+rst)
						good = true
						break
					}
				}
				if !good {
					s.dbgPrint(red+"failed to verify " + p.Endpoint+rst)
					badProx.Check(p)
					continue
				}

				switch p.Proto {
				case "4":
					s.Socks4 <- p
				case "4a":
					s.Socks4a <- p
				case "5":
					s.Socks5 <- p
				}
			}
		})
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}
