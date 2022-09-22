package prox5

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"git.tcp.direct/kayos/prox5/internal/pools"
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
	parent *Swamp
}

// Printf is used to handle socks server logging.
func (s SocksLogger) Printf(format string, a ...interface{}) {
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString(fmt.Sprintf(format, a...))
	s.parent.dbgPrint(buf)
}

type basicPrinter struct{}

func (b *basicPrinter) Print(str string) {
	println("[prox5] " + str)
}

func (b *basicPrinter) Printf(format string, items ...any) {
	println(fmt.Sprintf("prox5: "+format, items))
}

// DebugEnabled returns the current state of our debug switch.
func (p5 *Swamp) DebugEnabled() bool {
	debugHardLock.RLock()
	defer debugHardLock.RUnlock()
	return atomic.CompareAndSwapUint32(debugStatus, debugEnabled, debugEnabled)
}

// EnableDebug enables printing of verbose messages during operation
func (p5 *Swamp) EnableDebug() {
	atomic.StoreUint32(debugStatus, debugEnabled)
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (p5 *Swamp) DisableDebug() {
	atomic.StoreUint32(debugStatus, debugDisabled)
}

func simpleString(s string) *strings.Builder {
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString(s)
	return buf
}

func (p5 *Swamp) dbgPrint(builder *strings.Builder) {
	defer pools.DiscardBuffer(builder)
	if !p5.DebugEnabled() {
		return
	}
	p5.DebugLogger.Print(builder.String())
	return
}

func (p5 *Swamp) msgUnableToReach(socksString, target string, err error) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("unable to reach ")
	if p5.swampopt.redact {
		buf.WriteString("[redacted]")
	} else {
		buf.WriteString(target)
	}
	buf.WriteString(" with ")
	buf.WriteString(socksString)
	if !p5.swampopt.redact {
		buf.WriteString(": ")
		buf.WriteString(err.Error())
	}
	buf.WriteString(", cycling...")
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgUsingProxy(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("MysteryDialer using socks: ")
	buf.WriteString(socksString)
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgFailedMiddleware(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("failed middleware check, ")
	buf.WriteString(socksString)
	buf.WriteString(", cycling...")
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgTry(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("try dial with: ")
	buf.WriteString(socksString)
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgCantGetLock(socksString string, putback bool) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("can't get lock for ")
	buf.WriteString(socksString)
	if putback {
		buf.WriteString(", putting back in queue")
	}
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgGotLock(socksString string) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("got lock for ")
	buf.WriteString(socksString)
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgChecked(sock *Proxy, success bool) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	if success {
		buf.WriteString("verified ")
		buf.WriteString(sock.Endpoint)
		buf.WriteString(" as ")
		buf.WriteString(sock.protocol.Get().String())
		buf.WriteString(" proxy")
		p5.dbgPrint(buf)
		return
	}
	buf.WriteString("failed to verify: ")
	buf.WriteString(sock.Endpoint)
	p5.dbgPrint(buf)
}

func (p5 *Swamp) msgBadProxRate(sock *Proxy) {
	if !p5.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("badProx ratelimited: ")
	buf.WriteString(sock.Endpoint)
	p5.dbgPrint(buf)
}
