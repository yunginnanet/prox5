package pxndscvm

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
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

var (
	useProx        *rl.Limiter
	badProx        *rl.Limiter
	myipsites      = []string{"https://tcp.ac/ip", "https://vx-underground.org/ip", "https://wtfismyip.com/text"}
	// GoodProxies is a constant stream of verified proxies
	GoodProxies    chan string
	// PendingProxies is a constant stream of proxies to be verified
	PendingProxies chan string
	// Validated is a simple ticker to keep track of proxies we have verified since we started
	Validated      = 0
	// Birthday represents the time we started checking proxies
	Birthday       time.Time
	// UserAgents contains a list of UserAgents to use while making proxied requests, this should be supplied via SetUserAgents
	UserAgents []string

	prox []string

	mu *sync.RWMutex
)

// Proxy represents and individual proxy
type Proxy struct {
	s string
}

// UniqueKey is an implementation of the Identity interface from Rate5
func (p *Proxy) UniqueKey() string {
	return p.s
}

func init() {
	mu = &sync.RWMutex{}
	useProx = rl.NewLimiter(60, 2)
	badProx = rl.NewStrictLimiter(30, 50)
	PendingProxies = make(chan string, 2000)
	GoodProxies = make(chan string, 10000)
}

// LoadProxies loads proxies from a given seed file and randomly feeds them to the workers.
// This fucntion has no real error handling, if the file can't be opened it's gonna straight up panic.
// TODO: make it more gooder.
func LoadProxies(seedFile string) {
	f, err := os.Open(seedFile)
	if err != nil {
		panic(err)
	}

	scan := bufio.NewScanner(f)
	go tossUp()
	for scan.Scan() {
		prox = append(prox, scan.Text())
	}
	f.Close()
	for {
		select {
		case PendingProxies <- RandStrChoice(prox):
			//
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func SetUserAgents(uagents []string) {
	// mutex lock so that RLock during proxy checking will block while we change this value
	mu.Lock()
	defer mu.Unlock()
	UserAgents = uagents
}

func proxyGETRequest(sock string) (string, error) {
	mu.RLock()
	defer mu.RUnlock()
	req, err := http.NewRequest("GET", RandStrChoice(myipsites), bytes.NewBuffer([]byte("")))
	if err != nil {
		return "", err
	}

	headers := make(map[string]string)
	// headers["Host"] = "wtfismyip.com"
	headers["User-Agent"] = RandStrChoice(UserAgents)
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

func singleProxyCheck(sock string) bool {
	mu.RLock()
	defer mu.RUnlock()
	if _, err := net.DialTimeout("tcp", sock, 8*time.Second); err != nil {
		badProx.Check(&Proxy{s: sock})
		return false
	}
	resp, err := proxyGETRequest(sock)
	if err != nil {
		badProx.Check(&Proxy{s: sock})
		return false
	}
	if newip := net.ParseIP(resp); newip == nil {
		badProx.Check(&Proxy{s: sock})
		return false
	}
	Validated++
	log.Debug().Str("socks5", resp).Int("count", Validated).Msg("proxy validated")
	return true
}

func tossUp() {
	Birthday = time.Now()
	panicHandler := func(p interface{}) {
		log.Error().Interface("panic", p).Msg("Task panicked")
	}
	pool := pond.New(100, 10000, pond.MinWorkers(100), pond.PanicHandler(panicHandler))
	for {
		pool.Submit(func() {
			for {
				sock := <-PendingProxies
				if useProx.Check(&Proxy{s: sock}) {
					continue
				}
				if badProx.Peek(&Proxy{s: sock}) {
					continue
				}
				if singleProxyCheck(sock) {
					GoodProxies <- sock
					return
				}
			}
		})
		time.Sleep(time.Duration(10) * time.Millisecond)
	}
}

func GetProxy() string {
	for {
		select {
		case sock := <-GoodProxies:
			return sock
		default:
			time.Sleep(250 * time.Millisecond)
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
