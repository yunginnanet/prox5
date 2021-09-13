package pxndscvm

import (
	"fmt"
	"sync"
	"time"

	"github.com/alitto/pond"
	rl "github.com/yunginnanet/Rate5"
)

// SwampStatus represents the current state of our Swamp.
type SwampStatus int

const (
	// Running means the proxy pool is currently taking in proxys and validating them, and is available to dispense proxies.
	Running SwampStatus = iota
	// Paused means the proxy pool has been with Swamp.Pause() and may be resumed with Swamp.Resume()
	Paused
)

// Swamp represents a proxy pool
type Swamp struct {
	// Socks5 is a constant stream of verified Socks5 proxies
	Socks5 chan Proxy
	// Socks4 is a constant stream of verified Socks4 proxies
	Socks4 chan Proxy
	// Socks4a is a constant stream of verified Socks5 proxies
	Socks4a chan Proxy

	// Stats holds the Statistics for our swamp
	Stats *Statistics

	Status SwampStatus

	// Pending is a constant stream of proxy strings to be verified
	Pending chan string

	quit     chan bool
	scvm     []string
	pool     *pond.WorkerPool
	swampopt *SwampOptions
	started  bool
	mu       *sync.RWMutex
}

var (
	defaultStaleTime = 1 * time.Hour
	defWorkers       = 100
	// Note: I've chosen to use https here exclusively assuring all validated proxies are SSL capable.
	defaultChecks = []string{"https://wtfismyip.com/text", "https://myexternalip.com/raw", "https://ipinfo.io/ip", "https://api.ipify.org", "https://icanhazip.com/", "https://ifconfig.me/ip", "https://www.trackip.net/ip", "https://checkip.amazonaws.com/"}
)

func defOpt() *SwampOptions {
	return &SwampOptions{
		UserAgents:        DefaultUserAgents,
		CheckEndpoints:    defaultChecks,
		Stale:             defaultStaleTime,
		MaxWorkers:        defWorkers,
		ValidationTimeout: 5,
		Debug:             false,
	}
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
	// CheckEndpoints includes web services that respond with (just) the WAN IP of the connection for validation purposes
	CheckEndpoints []string
	// MaxWorkers determines the maximum amount of workers used for checking proxies
	MaxWorkers int
	// ValidationTimeout defines the timeout (in seconds) for proxy validation operations.
	// This will apply for both the initial quick check (dial), and the second check (HTTP GET).
	ValidationTimeout int
}

var (
	useProx *rl.Limiter
	badProx *rl.Limiter
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
func (p Proxy) UniqueKey() string {
	return p.Endpoint
}

func init() {
	useProx = rl.NewLimiter(60, 2)
	badProx = rl.NewStrictLimiter(30, 50)
}

// NewDefaultSwamp returns a Swamp with basic options.
func NewDefaultSwamp() *Swamp {
	s := &Swamp{
		Socks5:  make(chan Proxy, 1000),
		Socks4:  make(chan Proxy, 1000),
		Socks4a: make(chan Proxy, 1000),
		Pending: make(chan string, 500),

		Stats: &Statistics{
			Valid4:    0,
			Valid4a:   0,
			Valid5:    0,
			Dispensed: 0,
			Birthday:  time.Now(),
			mu:        &sync.Mutex{},
		},

		quit:     make(chan bool),
		swampopt: defOpt(),
		mu:       &sync.RWMutex{},
	}

	s.pool = pond.New(s.swampopt.MaxWorkers, 10000, pond.PanicHandler(func(p interface{}) {
		fmt.Println("WORKER PANIC! ", p)
	}))

	return s
}
