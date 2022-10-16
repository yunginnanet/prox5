package prox5

import (
	"fmt"
	"sync"
	"sync/atomic"

	"git.tcp.direct/kayos/common/pool"
)

var (
	debugStatus   *uint32
	debugHardLock = &sync.RWMutex{}
)

func init() {
	dd := debugDisabled
	debugStatus = &dd
}

const (
	debugEnabled uint32 = iota
	debugDisabled
)

type SocksLogger struct {
	parent *ProxyEngine
}

// Printf is used to handle socks server logging.
func (s SocksLogger) Printf(format string, a ...interface{}) {
	buf := strs.Get()
	buf.MustWriteString(fmt.Sprintf(format, a...))
	s.parent.dbgPrint(buf)
}

type basicPrinter struct{}

func (b *basicPrinter) Print(str string) {
	if !useDebugChannel {
		println("[prox5] " + str)
	} else {
		debugChan <- str
	}
}

func (b *basicPrinter) Printf(format string, items ...any) {
	str := fmt.Sprintf("[prox5] "+format, items)
	if !useDebugChannel {
		println(str)
	} else {
		debugChan <- str
	}
}

// DebugEnabled returns the current state of our debug switch.
func (p5 *ProxyEngine) DebugEnabled() bool {
	debugHardLock.RLock()
	defer debugHardLock.RUnlock()
	return atomic.CompareAndSwapUint32(debugStatus, debugEnabled, debugEnabled)
}

// EnableDebug enables printing of verbose messages during operation
func (p5 *ProxyEngine) EnableDebug() {
	atomic.StoreUint32(debugStatus, debugEnabled)
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (p5 *ProxyEngine) DisableDebug() {
	atomic.StoreUint32(debugStatus, debugDisabled)
}

func simpleString(s string) *pool.String {
	buf := strs.Get()
	buf.MustWriteString(s)
	return buf
}

func (p5 *ProxyEngine) dbgPrint(builder *pool.String) {
	defer strs.MustPut(builder)
	if !p5.DebugEnabled() {
		return
	}
	p5.DebugLogger.Print(builder.String())
	return
}

func (p5 *ProxyEngine) msgUnableToReach(socksString, target string, err error) {
	if !p5.DebugEnabled() {
		return
	}
	buf := strs.Get()
	buf.MustWriteString("unable to reach ")
	if p5.opt.redact {
		buf.MustWriteString("[redacted]")
	} else {
		buf.MustWriteString(target)
	}
	buf.MustWriteString(" with ")
	buf.MustWriteString(socksString)
	if !p5.opt.redact {
		buf.MustWriteString(": ")
		buf.MustWriteString(err.Error())
	}
	buf.MustWriteString(", cycling...")
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgUsingProxy(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	buf := strs.Get()
	if p5.GetDebugRedactStatus() {
		socksString = "(redacted)"
	}
	buf.MustWriteString("MysteryDialer using socks: ")
	buf.MustWriteString(socksString)
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgFailedMiddleware(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	buf := strs.Get()
	buf.MustWriteString("failed middleware check, ")
	buf.MustWriteString(socksString)
	buf.MustWriteString(", cycling...")
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgTry(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	if p5.GetDebugRedactStatus() {
		socksString = "(redacted)"
	}
	buf := strs.Get()
	buf.MustWriteString("try dial with: ")
	buf.MustWriteString(socksString)
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgCantGetLock(socksString string, putback bool) {
	if !p5.DebugEnabled() {
		return
	}
	if p5.GetDebugRedactStatus() {
		socksString = "(redacted)"
	}
	buf := strs.Get()
	buf.MustWriteString("can't get lock for ")
	buf.MustWriteString(socksString)
	if putback {
		buf.MustWriteString(", putting back in queue")
	}
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgGotLock(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	if p5.GetDebugRedactStatus() {
		socksString = "(redacted)"
	}
	buf := strs.Get()
	buf.MustWriteString("got lock for ")
	buf.MustWriteString(socksString)
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgChecked(sock *Proxy, success bool) {
	if !p5.DebugEnabled() {
		return
	}
	pstr := sock.Endpoint
	if p5.GetDebugRedactStatus() {
		pstr = "(redacted)"
	}
	buf := strs.Get()
	if !success {
		buf.MustWriteString("failed to verify: ")
		buf.MustWriteString(pstr)
		p5.dbgPrint(buf)
		return
	}
	buf.MustWriteString("verified ")
	buf.MustWriteString(pstr)
	buf.MustWriteString(" as ")
	buf.MustWriteString(sock.protocol.Get().String())
	buf.MustWriteString(" proxy")
	p5.dbgPrint(buf)
}

func (p5 *ProxyEngine) msgBadProxRate(sock *Proxy) {
	if !p5.DebugEnabled() {
		return
	}
	sockString := sock.Endpoint
	if p5.GetDebugRedactStatus() {
		sockString = "(redacted)"
	}
	buf := strs.Get()
	buf.MustWriteString("badProx ratelimited: ")
	buf.MustWriteString(sockString)
	p5.dbgPrint(buf)
}

// ------------

var (
	debugChan       chan string
	useDebugChannel bool
)

// DebugChannel will return a channel which will receive debug messages once debug is enabled.
// This will alter the flow of debug messages, they will no longer print to console, they will be pushed into this channel.
// Make sure you pull from the channel eventually to avoid build up of blocked goroutines.
//
// Deprecated: use DebugLogger instead. This will be removed in a future version.
func (p5 *ProxyEngine) DebugChannel() chan string {
	debugChan = make(chan string, 100)
	useDebugChannel = true
	return debugChan
}
