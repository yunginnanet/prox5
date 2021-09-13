package pxndscvm

import (
	"sync"
	"time"

	rl "github.com/yunginnanet/Rate5"
)

// Swamp represents a proxy pool
type Swamp struct {
	// Socks5 is a constant stream of verified Socks5 proxies
	Socks5 chan *Proxy
	// Socks4 is a constant stream of verified Socks4 proxies
	Socks4 chan *Proxy
	// Socks4a is a constant stream of verified Socks5 proxies
	Socks4a chan *Proxy

	// Stats holds the Statistics for our swamp
	Stats *Statistics

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
	// DefaultUserAgents representts our default list of useragents

	// DefaultStaleTime represents our default setting for stale proxies
	DefaultStaleTime = time.Duration(1) * time.Hour
)

func defOpt() *SwampOptions {
	return &SwampOptions{UserAgents: DefaultUserAgents, Stale: DefaultStaleTime}
}

// SwampOptions holds our configuration for Swamp instances
type SwampOptions struct {
	// UserAgents contains a list of UserAgents to be randomly drawn from for proxied requests, this should be supplied via SetUserAgents
	UserAgents []string
	// Stale is the amount of time since verification that qualifies a proxy going stale.
	// if a stale proxy is drawn during the use of our getter functions, it will be skipped.
	Stale time.Duration
	// Debug when enabled will print results as they come in
	Debug bool
}

var (
	useProx   *rl.Limiter
	badProx   *rl.Limiter
	myipsites = []string{"https://tcp.ac/ip", "https://vx-underground.org/ip", "https://wtfismyip.com/text"}
)

// Proxy represents an individual proxy
type Proxy struct {
	// Endpoint is the address:port of the proxy that we connect to
	Endpoint string
	// ProxiedIP is the address that we end up having when making proxied requests through this proxy
	ProxiedIP string
	// Proto is the version/Protocol (currently SOCKS* only) of the proxy
	Proto string
	// Verified is the time this proxy was last verified working
	Verified time.Time
}

// UniqueKey is an implementation of the Identity interface from Rate5
func (p *Proxy) UniqueKey() string {
	return p.Endpoint
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

		Stats: &Statistics{
			validated4: 0,
			validated4a: 0,
			validated5: 0,
			mu: &sync.Mutex{},
		},

		Dispensed: 0,
		Birthday:  time.Now(),

		swampopt: defOpt(),
		mu:       &sync.RWMutex{},
	}
}
