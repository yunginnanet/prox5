package prox5

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	rl "github.com/yunginnanet/Rate5"

	"git.tcp.direct/kayos/prox5/internal/scaler"
	"git.tcp.direct/kayos/prox5/logger"
)

type ProxyChannels struct {
	// SOCKS5 is a constant stream of verified SOCKS5 proxies
	SOCKS5 chan *Proxy
	// SOCKS4 is a constant stream of verified SOCKS4 proxies
	SOCKS4 chan *Proxy
	// SOCKS4a is a constant stream of verified SOCKS5 proxies
	SOCKS4a chan *Proxy
	// HTTP is a constant stream of verified SOCKS5 proxies
	HTTP chan *Proxy
}

// ProxyEngine represents a proxy pool
type ProxyEngine struct {
	Valids      ProxyChannels
	DebugLogger logger.Logger

	// stats holds the Statistics for ProxyEngine
	stats *Statistics

	Status uint32

	// Pending is a constant stream of proxy strings to be verified
	Pending chan *Proxy

	// see: https://pkg.go.dev/github.com/yunginnanet/Rate5
	useProx *rl.Limiter
	badProx *rl.Limiter

	dispenseMiddleware func(*Proxy) (*Proxy, bool)

	conCtx    context.Context
	killConns context.CancelFunc
	ctx       context.Context
	quit      context.CancelFunc

	proxyMap proxyMap

	// reaper sync.Pool

	mu             *sync.RWMutex
	pool           *ants.Pool
	scaler         *scaler.AutoScaler
	opt            *config
	runningdaemons int32
	conductor      chan bool
}

var (
	defaultStaleTime   = 30 * time.Minute
	defaultWorkerCount = 20
	defaultBailout     = 20
	defaultRemoveAfter = 25
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

// Returns a pointer to our default options (modified and accessed later through concurrent safe getters and setters)
func defOpt() *config {
	sm := &config{
		useProxConfig: defaultUseProxyRatelimiter,
		badProxConfig: defaultBadProxyRateLimiter,

		checkEndpoints: defaultChecks,
		userAgents:     defaultUserAgents,
		RWMutex:        &sync.RWMutex{},
		removeafter:    defaultRemoveAfter,
		recycle:        true,
		debug:          true,
		dialerBailout:  defaultBailout,
		stale:          defaultStaleTime,
		maxWorkers:     defaultWorkerCount,
		redact:         true,
	}
	sm.validationTimeout = time.Duration(18) * time.Second
	sm.serverTimeout = time.Duration(180) * time.Second
	return sm
}

// config holds our configuration for ProxyEngine instances.
// This is implemented as a pointer, and should be interacted with via the setter and getter functions.
type config struct {
	// stale is the amount of time since verification that qualifies a proxy going stale.
	// if a stale proxy is drawn during the use of our getter functions, it will be skipped.
	stale time.Duration
	// userAgents contains a list of userAgents to be randomly drawn from for proxied requests, this should be supplied via SetUserAgents
	userAgents []string
	// debug when enabled will print results as they come in
	debug bool
	// checkEndpoints includes web services that respond with (just) the WAN IP of the connection for validation purposes
	checkEndpoints []string
	// maxWorkers determines the maximum amount of workers used for checking proxies
	maxWorkers int
	// validationTimeout defines the timeout for proxy validation operations.
	// This will apply for both the initial quick check (dial), and the second check (HTTP GET).
	validationTimeout time.Duration
	// serverTimeout defines the timeout for outgoing connections made with the MysteryDialer.
	serverTimeout time.Duration
	// dialerBailout defines the amount of times a dial atttempt can fail before giving up and returning an error.
	dialerBailout int
	// redact when enabled will redact the target string from the debug output
	redact bool
	// recycle determines whether or not we recycle proxies pack into the pending channel after we dispense them
	recycle bool
	// remove proxy from recycling after being marked bad this many times
	removeafter int
	// shuffle determines whether or not we shuffle proxies before we validate and dispense them.
	// shuffle bool

	// TODO: make getters and setters for these
	useProxConfig rl.Policy
	badProxConfig rl.Policy

	*sync.RWMutex
}

// NewDefaultSwamp returns a new ProxyEngine instance.
//
// Deprecated: use NewProxyEngine instead.
func NewDefaultSwamp() *Swamp {
	return &Swamp{NewProxyEngine()}
}

// Swamp is a deprecated alias for ProxyEngine
//
// Deprecated: use ProxyEngine instead.
type Swamp struct {
	*ProxyEngine
}

// NewProxyEngine returns a ProxyEngine with default options.
// After calling this you may use the various "setters" to change the options before calling ProxyEngine.Start().
func NewProxyEngine() *ProxyEngine {
	pe := &ProxyEngine{
		stats:       &Statistics{birthday: time.Now()},
		DebugLogger: &basicPrinter{},

		opt: defOpt(),

		conductor: make(chan bool),
		mu:        &sync.RWMutex{},
		Status:    uint32(StateNew),
	}

	stats := []int64{pe.stats.Valid4, pe.stats.Valid4a, pe.stats.Valid5, pe.stats.ValidHTTP, pe.stats.Dispensed}
	for i := range stats {
		atomic.StoreInt64(&stats[i], 0)
	}

	chans := []*chan *Proxy{&pe.Valids.SOCKS5, &pe.Valids.SOCKS4, &pe.Valids.SOCKS4a, &pe.Valids.HTTP, &pe.Pending}
	for _, c := range chans {
		*c = make(chan *Proxy, 10000)
	}

	pe.dispenseMiddleware = func(p *Proxy) (*Proxy, bool) {
		return p, true
	}
	pe.ctx, pe.quit = context.WithCancel(context.Background())
	pe.conCtx, pe.killConns = context.WithCancel(context.Background())
	pe.proxyMap = newProxyMap(pe)

	atomic.StoreUint32(&pe.Status, uint32(StateNew))
	atomic.StoreInt32(&pe.runningdaemons, 0)

	pe.useProx = rl.NewCustomLimiter(pe.opt.useProxConfig)
	pe.badProx = rl.NewCustomLimiter(pe.opt.badProxConfig)

	var err error
	pe.pool, err = ants.NewPool(pe.opt.maxWorkers, ants.WithOptions(ants.Options{
		ExpiryDuration: 2 * time.Minute,
		PanicHandler:   pe.pondPanic,
	}))

	pe.scaler = scaler.NewAutoScaler(pe.opt.maxWorkers+100, 10)

	if err != nil {
		buf := strs.Get()
		buf.MustWriteString("CRITICAL: ")
		buf.MustWriteString(err.Error())
		pe.dbgPrint(buf)
		panic(err)
	}

	return pe
}

func newProxyMap(pe *ProxyEngine) proxyMap {
	return proxyMap{
		plot:   make(map[string]*Proxy),
		mu:     &sync.RWMutex{},
		parent: pe,
	}
}

func (p5 *ProxyEngine) pondPanic(p interface{}) {
	panic(p)
	// pe.dbgPrint("Worker panic: " + fmt.Sprintf("%v", p))
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
