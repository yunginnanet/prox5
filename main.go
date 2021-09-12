package pxndscvm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alitto/pond"
	"github.com/rs/zerolog/log"
	rl "github.com/yunginnanet/Rate5"
	"golang.org/x/net/proxy"
	"h12.io/socks"
)

// Swamp represents a proxy pool
type Swamp struct {
	// Socks5 is a constant stream of verified Socks5 proxies
	Socks5 chan *Proxy
	// Socks4 is a constant stream of verified Socks4 proxies
	Socks4 chan *Proxy
	// Socks4a is a constant stream of verified Socks5 proxies
	Socks4a chan *Proxy

	// Validated is a simple ticker to keep track of proxies we have verified since we started
	Validated int
	// Dispensed is a simple ticker to keep track of proxies dispensed via our getters
	Dispensed int

	// Pending is a constant stream of proxy strings to be verified
	Pending chan string

	// Birthday represents the time we started checking proxies with this pool
	Birthday time.Time

	scvm     []string
	swampopt *SwampOptions
	mu       *sync.RWMutex
}

var (
	DefaultUserAgents = []string{"asdf"}
	DefaultStaleTime  = time.Duration(1) * time.Hour
)

func defOpt() *SwampOptions {
	return &SwampOptions{UserAgents: DefaultUserAgents, Stale: DefaultStaleTime}
}

type SwampOptions struct {
	// UserAgents contains a list of UserAgents to be randomly drawn from for proxied requests, this should be supplied via SetUserAgents
	UserAgents []string
	// Stale is the amount of time since verification that qualifies a proxy going stale.
	//		if a stale proxy is drawn during the use of our getter functions, it will be skipped.
	Stale time.Duration
}

var (
	useProx   *rl.Limiter
	badProx   *rl.Limiter
	myipsites = []string{"https://tcp.ac/ip", "https://vx-underground.org/ip", "https://wtfismyip.com/text"}
)

// Proxy represents and individual proxy
type Proxy struct {
	endpoint string
	/*
		SOCKS4 = iota
		SOCKS4A
		SOCKS5
	*/
	proto int

	// timestamp of verificaiton
	verified time.Time
}

// UniqueKey is an implementation of the Identity interface from Rate5
func (p *Proxy) UniqueKey() string {
	return p.endpoint
}

func init() {
	useProx = rl.NewLimiter(60, 2)
	badProx = rl.NewStrictLimiter(30, 50)
}

// NewDefaultSwamp returns a Swamp with basic options.
func NewDefaultSwamp() *Swamp {
	return &Swamp{
		Socks5:  make(chan *Proxy, 500),
		Socks4:  make(chan *Proxy, 500),
		Socks4a: make(chan *Proxy, 500),
		Pending: make(chan string, 1000),

		Validated: 0,
		Dispensed: 0,
		Birthday:  time.Now(),

		swampopt: defOpt(),
		mu:       &sync.RWMutex{},
	}
}

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

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (s *Swamp) AddUserAgents(uagents []string) {
	// mutex lock so that RLock during proxy checking will block while we change this value
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.UserAgents = append(s.swampopt.UserAgents, uagents...)
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (s *Swamp) SetUserAgents(uagents []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.UserAgents = append(s.swampopt.UserAgents, uagents...)
}

// RandomUserAgent retrieves a random user agent from our list in string form
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return RandStrChoice(s.swampopt.UserAgents)
}


// MysteryDial will return a dialer that will use a different proxy for every request.
func (s *Swamp) MysteryDial(ctx context.Context, network, addr string) (net.Conn, error) {
	var sock = ""
	var ver string
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
		candidate := getProxy()
		switch {
		// if since its been validated it's ended up failing so much that its on our bad list then skip it
		case badProx.Peek(candidate):
			fallthrough
		// if we've been checking or using this too often recently then skip it
		case useProx.Check(candidate):
			fallthrough
		case time.Since(candidate.verified) > s.swampopt.Stale:
			continue
		default:
			sock = candidate.endpoint
			break
		}
		if sock != "" {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var dialSocks = socks.Dial("socks" + ver + "://" + sock + "?timeout=3s")


}

func (s *Swamp) proxyGETRequest() (string, error) {
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

	var dialSocks = socks.Dial("socks5://" + sock + "?timeout=3s")
	var client *http.Client

	if sock == "none" {
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

func (s *Swamp) singleProxyCheck(sock string) (*Proxy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	new := &Proxy{endpoint: sock}

	if _, err := net.DialTimeout("tcp", sock, 8*time.Second); err != nil {
		badProx.Check(new)
		return new, nil
	}

	resp, err := s.proxyGETRequest()
	if err != nil {
		badProx.Check(&Proxy{endpoint: sock})
		return nil, err
	}
	if newip := net.ParseIP(resp); newip == nil {
		badProx.Check(&Proxy{endpoint: sock})
		return nil, errors.New("nil response from http request")
	}
	s.Validated++
	log.Debug().Str("socks5", resp).Int("count", s.Validated).Msg("proxy validated")
	return new, nil
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
				sock := <- s.Pending
				if useProx.Check(&Proxy{endpoint: sock}) {
					continue
				}
				if badProx.Peek(&Proxy{endpoint: sock}) {
					continue
				}
				if s.singleProxyCheck(sock) {
					GoodProxies <- &Proxy{endpoint: sock}
					return
				}
			}
		})
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func getProxy() *Proxy {
	for {
		select {
		case sock := <-GoodProxies:
			return sock
		}
	}
}

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
func (s *Swamp) Socks5Str() string {
	for {
		select {
		case sock := <-s.Socks5:
			return sock.endpoint
		}
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
func (s *Swamp) Socks4Str() string {
	for {
		select {
		case sock := <-s.Socks4:
			return sock.endpoint
		}
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
func (s *Swamp) Socks4aStr() string {
	for {
		select {
		case sock := <-s.Socks4a:
			return sock.endpoint
		}
	}
}

// RandStrChoice returns a random element from the given string slice
func RandStrChoice(choices []string) string {
	strlen := len(choices)
	n := uint32(0)
	if strlen > 0 {
		n = GetRandomUint32() % uint32(strlen)
	}
	return choices[n]
}

// GetRandomUint32 retrieves a cryptographically sound random 32 bit unsigned little endian integer
func GetRandomUint32() uint32 {
	b := make([]byte, 8192)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(b)
}
