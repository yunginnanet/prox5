package prox5

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/panjf2000/ants/v2"
	rl "github.com/yunginnanet/Rate5"

	"git.tcp.direct/kayos/prox5/internal/scaler"
	"git.tcp.direct/kayos/prox5/logger"
)

type proxyList struct {
	*list.List
	*sync.RWMutex
}

func (pl proxyList) add(p *Proxy) {
	pl.Lock()
	defer pl.Unlock()
	pl.PushBack(p)
}

func (pl proxyList) pop() *Proxy {
	pl.Lock()
	if pl.Len() < 1 {
		pl.Unlock()
		return nil
	}
	p := pl.Remove(pl.Front()).(*Proxy)
	pl.Unlock()
	return p
}

// ProxyChannels will likely be unexported in the future.
type ProxyChannels struct {
	// SOCKS5 is a constant stream of verified SOCKS5 proxies
	SOCKS5 proxyList
	// SOCKS4 is a constant stream of verified SOCKS4 proxies
	SOCKS4 proxyList
	// SOCKS4a is a constant stream of verified SOCKS5 proxies
	SOCKS4a proxyList
	// HTTP is a constant stream of verified SOCKS5 proxies
	HTTP proxyList
}

// Slice returns a slice of all proxyLists in ProxyChannels, note that HTTP is not included.
func (pc ProxyChannels) Slice() []*proxyList {
	lists := []*proxyList{&pc.SOCKS5, &pc.SOCKS4, &pc.SOCKS4a}
	entropy.GetOptimizedRand().Shuffle(3, func(i, j int) {
		lists[i], lists[j] = lists[j], lists[i]
	})
	return lists
}

// ProxyEngine represents a proxy pool
type ProxyEngine struct {
	Valids      ProxyChannels
	DebugLogger logger.Logger

	// stats holds the Statistics for ProxyEngine
	stats *Statistics

	Status uint32

	// Pending is a constant stream of proxy strings to be verified
	Pending proxyList

	// see: https://pkg.go.dev/github.com/yunginnanet/Rate5
	useProx *rl.Limiter
	badProx *rl.Limiter

	dispenseMiddleware func(*Proxy) (*Proxy, bool)

	conCtx    context.Context
	killConns context.CancelFunc
	ctx       context.Context
	quit      context.CancelFunc

	httpOptsDirty *atomic.Bool
	httpClients   *sync.Pool

	proxyMap proxyMap

	// reaper sync.Pool

	recycleMu *sync.Mutex
	mu        *sync.RWMutex
	pool      *ants.Pool

	scaler     *scaler.AutoScaler
	scaleTimer *time.Ticker

	recycleTimer *time.Ticker

	lastBadProxAnnnounced *atomic.Value

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
		redact:         false,
		tlsVerify:      false,
		shuffle:        true,
	}
	sm.validationTimeout = time.Duration(9) * time.Second
	sm.serverTimeout = time.Duration(15) * time.Second
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
	// serverTimeout defines the timeout for outgoing connections made with the mysteryDialer.
	serverTimeout time.Duration
	// dialerBailout defines the amount of times a dial atttempt can fail before giving up and returning an error.
	dialerBailout int
	// redact when enabled will redact the target string from the debug output
	redact bool
	// recycle determines whether or not we recycle proxies pack into the pending channel after we dispense them
	recycle bool
	// remove proxy from recycling after being marked bad this many times
	removeafter int
	// shuffle determines whether or not we shuffle proxies when we recycle them.
	shuffle bool
	// tlsVerify determines whether or not we verify the TLS certificate of the endpoints the http client connects to.
	tlsVerify bool

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
	p5 := &ProxyEngine{
		stats:       &Statistics{birthday: time.Now()},
		DebugLogger: &basicPrinter{},

		opt:                   defOpt(),
		lastBadProxAnnnounced: &atomic.Value{},

		conductor:     make(chan bool),
		mu:            &sync.RWMutex{},
		recycleMu:     &sync.Mutex{},
		httpOptsDirty: &atomic.Bool{},
		Status:        uint32(stateNew),
	}

	p5.lastBadProxAnnnounced.Store("")
	p5.httpOptsDirty.Store(false)
	p5.httpClients = &sync.Pool{New: func() interface{} { return p5.newHTTPClient() }}

	stats := []int64{p5.stats.Valid4, p5.stats.Valid4a, p5.stats.Valid5, p5.stats.ValidHTTP, p5.stats.Dispensed}
	for i := range stats {
		atomic.StoreInt64(&stats[i], 0)
	}

	lists := []*proxyList{&p5.Valids.SOCKS5, &p5.Valids.SOCKS4, &p5.Valids.SOCKS4a, &p5.Valids.HTTP, &p5.Pending}
	for _, c := range lists {
		*c = proxyList{
			List:    &list.List{},
			RWMutex: &sync.RWMutex{},
		}
	}

	p5.dispenseMiddleware = func(p *Proxy) (*Proxy, bool) {
		return p, true
	}
	p5.ctx, p5.quit = context.WithCancel(context.Background())
	p5.conCtx, p5.killConns = context.WithCancel(context.Background())
	p5.proxyMap = newProxyMap(p5)

	atomic.StoreUint32(&p5.Status, uint32(stateNew))
	atomic.StoreInt32(&p5.runningdaemons, 0)

	p5.useProx = rl.NewCustomLimiter(p5.opt.useProxConfig)
	p5.badProx = rl.NewCustomLimiter(p5.opt.badProxConfig)

	var err error
	p5.pool, err = ants.NewPool(p5.opt.maxWorkers, ants.WithOptions(ants.Options{
		ExpiryDuration: 2 * time.Minute,
		PanicHandler:   p5.pondPanic,
	}))

	p5.scaler = scaler.NewAutoScaler(p5.opt.maxWorkers, p5.opt.maxWorkers+100, 50)
	p5.scaleTimer = time.NewTicker(1 * time.Second)
	p5.recycleTimer = time.NewTicker(500 * time.Millisecond)

	if err != nil {
		buf := strs.Get()
		buf.MustWriteString("CRITICAL: ")
		buf.MustWriteString(err.Error())
		p5.dbgPrint(buf)
		panic(err)
	}

	return p5
}

func newProxyMap(pe *ProxyEngine) proxyMap {
	return proxyMap{
		plot:   cmap.New[*Proxy](),
		parent: pe,
	}
}

func (p5 *ProxyEngine) pondPanic(p interface{}) {
	panic(p)
	// pe.dbgPrint("Worker panic: " + fmt.Sprintf("%v", p))
}

// defaultUserAgents is a small list of user agents to use during validation.
var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; rv109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.82",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 Edg/115.0.1901.188",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv109.0) Gecko/20100101 Firefox/114.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv102.0) Gecko/20100101 Firefox/102.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 Edg/115.0.1901.183",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.67",
	"Mozilla/5.0 (X11; Linux x86_64; rv109.0) Gecko/20100101 Firefox/114.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 OPR/100.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv102.0) Gecko/20100101 Firefox/102.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv109.0) Gecko/20100101 Firefox/114.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.79",
	"Mozilla/5.0 (X11; Linux x86_64; rv109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.75 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36 OPR/99.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.4 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; rv102.0) Gecko/20100101 Firefox/102.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; WOW64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.5666.197 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; rv109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (Windows NT 10.0; rv114.0) Gecko/20100101 Firefox/114.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv109.0) Gecko/20100101 Firefox/114.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.86",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv109.0) Gecko/20100101 Firefox/113.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 YaBrowser/23.5.4.674 Yowser/2.5 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
}
