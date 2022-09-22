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
	parent *ProxyEngine
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
func (pe *ProxyEngine) DebugEnabled() bool {
	debugHardLock.RLock()
	defer debugHardLock.RUnlock()
	return atomic.CompareAndSwapUint32(debugStatus, debugEnabled, debugEnabled)
}

// EnableDebug enables printing of verbose messages during operation
func (pe *ProxyEngine) EnableDebug() {
	atomic.StoreUint32(debugStatus, debugEnabled)
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (pe *ProxyEngine) DisableDebug() {
	atomic.StoreUint32(debugStatus, debugDisabled)
}

func simpleString(s string) *strings.Builder {
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString(s)
	return buf
}

func (pe *ProxyEngine) dbgPrint(builder *strings.Builder) {
	defer pools.DiscardBuffer(builder)
	if !pe.DebugEnabled() {
		return
	}
	pe.DebugLogger.Print(builder.String())
	return
}

func (pe *ProxyEngine) msgUnableToReach(socksString, target string, err error) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("unable to reach ")
	if pe.swampopt.redact {
		buf.WriteString("[redacted]")
	} else {
		buf.WriteString(target)
	}
	buf.WriteString(" with ")
	buf.WriteString(socksString)
	if !pe.swampopt.redact {
		buf.WriteString(": ")
		buf.WriteString(err.Error())
	}
	buf.WriteString(", cycling...")
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgUsingProxy(socksString string) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("MysteryDialer using socks: ")
	buf.WriteString(socksString)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgFailedMiddleware(socksString string) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("failed middleware check, ")
	buf.WriteString(socksString)
	buf.WriteString(", cycling...")
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgTry(socksString string) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("try dial with: ")
	buf.WriteString(socksString)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgCantGetLock(socksString string, putback bool) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("can't get lock for ")
	buf.WriteString(socksString)
	if putback {
		buf.WriteString(", putting back in queue")
	}
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgGotLock(socksString string) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("got lock for ")
	buf.WriteString(socksString)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgChecked(sock *Proxy, success bool) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	if success {
		buf.WriteString("verified ")
		buf.WriteString(sock.Endpoint)
		buf.WriteString(" as ")
		buf.WriteString(sock.protocol.Get().String())
		buf.WriteString(" proxy")
		pe.dbgPrint(buf)
		return
	}
	buf.WriteString("failed to verify: ")
	buf.WriteString(sock.Endpoint)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgBadProxRate(sock *Proxy) {
	if !pe.DebugEnabled() {
		return
	}
	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("badProx ratelimited: ")
	buf.WriteString(sock.Endpoint)
	pe.dbgPrint(buf)
}
