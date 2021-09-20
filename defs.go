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
	Socks5 chan *Proxy
	// Socks4 is a constant stream of verified Socks4 proxies
	Socks4 chan *Proxy
	// Socks4a is a constant stream of verified Socks5 proxies
	Socks4a chan *Proxy

	// Stats holds the Statistics for our swamp
	Stats *Statistics

	Status SwampStatus

	// Pending is a constant stream of proxy strings to be verified
	Pending chan *Proxy

	// see: https://pkg.go.dev/github.com/yunginnanet/Rate5
	useProx *rl.Limiter
	badProx *rl.Limiter

	quit chan bool

	swampmap map[string]*Proxy

	pool           *pond.WorkerPool
	swampopt       *swampOptions
	runningdaemons int
	started        bool
	mu             *sync.RWMutex
}

var (
	defaultStaleTime = 1 * time.Hour
	defWorkers       = 100
	// Note: I've chosen to use https here exclusively assuring all validated proxies are SSL capable.
	defaultChecks = []string{
		"https://wtfismyip.com/text",
		"https://myexternalip.com/raw",
		"https://ipinfo.io/ip",
		"https://api.ipify.org/",
		"https://icanhazip.com/",
		"https://ifconfig.me/ip",
		"https://www.trackip.net/ip",
		"https://checkip.amazonaws.com/",
	}
)

// https://pkg.go.dev/github.com/yunginnanet/Rate5#Policy
var defUseProx = rl.Policy{
	Window: 60,
	Burst:  2,
}
var defBadProx = rl.Policy{
	Window: 60,
	Burst:  3,
}

// Returns a pointer to our default options (modified and accessed later through concurrent safe getters and setters)
func defOpt() *swampOptions {
	return &swampOptions{
		userAgents:     defaultUserAgents,
		CheckEndpoints: defaultChecks,
		stale:          defaultStaleTime,
		maxWorkers:     defWorkers,

		useProxConfig: defUseProx,
		badProxConfig: defBadProx,

		validationTimeout: 5,
		debug:             false,
	}
}

// swampOptions holds our configuration for Swamp instances.
// This is implemented as a pointer, and should be interacted with via the setter and getter functions.
type swampOptions struct {
	// stale is the amount of time since verification that qualifies a proxy going stale.
	// if a stale proxy is drawn during the use of our getter functions, it will be skipped.
	stale time.Duration
	// userAgents contains a list of userAgents to be randomly drawn from for proxied requests, this should be supplied via SetUserAgents
	userAgents []string
	// debug when enabled will print results as they come in
	debug bool
	// CheckEndpoints includes web services that respond with (just) the WAN IP of the connection for validation purposes
	CheckEndpoints []string
	// maxWorkers determines the maximum amount of workers used for checking proxies
	maxWorkers int
	// validationTimeout defines the timeout (in seconds) for proxy validation operations.
	// This will apply for both the initial quick check (dial), and the second check (HTTP GET).
	validationTimeout int

	// TODO: make getters and setters for these
	useProxConfig rl.Policy
	badProxConfig rl.Policy
}
const (
	stateUnlocked uint32 = iota
	stateLocked
)

// Proxy represents an individual proxy
type Proxy struct {
	// Endpoint is the address:port of the proxy that we connect to
	Endpoint string
	// ProxiedIP is the address that we end up having when making proxied requests through this proxy
	ProxiedIP string
	// Proto is the version/Protocol (currently SOCKS* only) of the proxy
	Proto string
	// LastVerified is the time this proxy was last verified working
	LastVerified time.Time

	lock uint32
}

// UniqueKey is an implementation of the Identity interface from Rate5
func (sock Proxy) UniqueKey() string {
	return sock.Endpoint
}

// NewDefaultSwamp returns a Swamp with basic options.
func NewDefaultSwamp() *Swamp {
	s := &Swamp{
		Socks5:  make(chan *Proxy, 10000),
		Socks4:  make(chan *Proxy, 10000),
		Socks4a: make(chan *Proxy, 10000),
		Pending: make(chan *Proxy, 10000),

		Stats: &Statistics{
			Valid4:    0,
			Valid4a:   0,
			Valid5:    0,
			Dispensed: 0,
			birthday:  time.Now(),
			mu:        &sync.Mutex{},
		},

		swampopt: defOpt(),

		quit: make(chan bool),
		mu:   &sync.RWMutex{},
	}

	s.swampmap = make(map[string]*Proxy)

	s.useProx = rl.NewCustomLimiter(s.swampopt.useProxConfig)
	s.badProx = rl.NewCustomLimiter(s.swampopt.badProxConfig)

	s.pool = pond.New(s.swampopt.maxWorkers, 10000, pond.PanicHandler(func(p interface{}) {
		fmt.Println("WORKER PANIC! ", p)
	}))

	return s
}

// defaultUserAgents is a small list of user agents to use during validation.
var defaultUserAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.12; rv:60.0) Gecko/20100101 Firefox/60.0",
	"Mozilla/5.0 (Windows NT 6.2; WOW64; rv:34.0) Gecko/20100101 Firefox/34.0",
	"Mozilla/5.0 (Windows NT 6.2; Win64; x64; rv:24.0) Gecko/20140419 Firefox/24.0 PaleMoon/24.5.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:44.0) Gecko/20100101 Firefox/44.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:49.0) Gecko/20100101 Firefox/49.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:55.0) Gecko/20100101 Firefox/55.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.11; rv:47.0) Gecko/20100101 Firefox/--.0",
	"Mozilla/5.0 (Windows NT 6.0; rv:19.0) Gecko/20100101 Firefox/19.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux i686; rv:45.0) Gecko/20100101 Firefox/45.0",
	"Mozilla/5.0 (Windows NT 6.0; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0",
	"Mozilla/5.0 (FreeBSD; Viera; rv:34.0) Gecko/20100101 Firefox/34.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.7; rv:20.0) Gecko/20100101 Firefox/20.0",
	"Mozilla/5.0 (Android 6.0; Mobile; rv:60.0) Gecko/20100101 Firefox/60.0",
	"Mozilla/5.0 (Windows NT 5.1; rv:37.0) Gecko/20100101 Firefox/37.0",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:35.0) Gecko/20100101 Firefox/35.0 evaliant",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:28.0) Gecko/20100101 Firefox/28.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:58.0) Gecko/20100101 Firefox/58.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:60.0) Gecko/20100101 Firefox/60.0",
	"Mozilla/5.0 (Windows NT 10.0; WOW64; rv:45.0) Gecko/20100101 Firefox/45.0",
	"Mozilla/5.0 (Windows NT 6.2; WOW64; rv:41.0) Gecko/20100101 Firefox/41.0",
}
