package pxndscvm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
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


// LoadProxyTXT loads proxies from a given seed file and randomly feeds them to the workers.
// This fucntion has no real error handling, if the file can't be opened it's gonna straight up panic.
// TODO: make it more gooder.
func (s *Swamp) LoadProxyTXT(seedFile string) {
	f, err := os.Open(seedFile)
	if err != nil {
		panic(err)
	}

	scan := bufio.NewScanner(f)
	go s.tossUp()
	for scan.Scan() {
		s.scvm = append(s.scvm, scan.Text())
	}
	f.Close()
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
		switch {
		// if since its been validated it's ended up failing so much that its on our bad list then skip it
		case badProx.Peek(candidate):
			fallthrough
		// if we've been checking or using this too often recently then skip it
		case useProx.Check(candidate):
			fallthrough
		case time.Since(candidate.Verified) > s.swampopt.Stale:
			continue
		default:
			sock = candidate
			break
		}
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

	defer resp.Body.Close()
	rbody, _ := ioutil.ReadAll(resp.Body)
	return string(rbody), nil
}

func (s *Swamp) singleProxyCheck(sock *Proxy) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := net.DialTimeout("tcp", sock.Endpoint, 8*time.Second); err != nil {
		badProx.Check(sock)
		return nil
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
	s.Validated++
	log.Debug().Str("socks5", resp).Int("count", s.Validated).Msg("proxy validated")
	return nil
}

func (s *Swamp) tossUp() {
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

				if useProx.Check(p) {
					continue
				}
				if badProx.Peek(p) {
					continue
				}
				if err := s.singleProxyCheck(p); err == nil {
					switch p.Proto {
					case "4":
						s.Socks4 <- p
					case "4a":
						s.Socks4a <- p
					case "5":
						s.Socks5 <- p
					}
				}
			}
		})
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}
