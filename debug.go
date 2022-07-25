package prox5

import (
	"fmt"
	"strings"
	"sync"
)

func init() {
	debugMutex = &sync.RWMutex{}
}

var (
	useDebugChannel = false
	debugChan       chan string
	debugMutex      *sync.RWMutex
)

type DebugPrinter interface {
	Print(str string)
	Printf(format string, items ...any)
}

type basicPrinter struct{}

func (b basicPrinter) Print(str string) {
	println("[prox5] " + str)
}

func (b basicPrinter) Printf(format string, items ...any) {
	println(fmt.Sprintf("prox5: "+format, items))
}

// DebugChannel will return a channel which will receive debug messages once debug is enabled.
// This will alter the flow of debug messages, they will no longer print to console, they will be pushed into this channel.
// Make sure you pull from the channel eventually to avoid build up of blocked goroutines.
func (pe *ProxyEngine) DebugChannel() chan string {
	debugChan = make(chan string, 1000000)
	useDebugChannel = true
	return debugChan
}

// DebugEnabled returns the current state of our debug switch.
func (pe *ProxyEngine) DebugEnabled() bool {
	return pe.swampopt.debug
}

// DisableDebugChannel redirects debug messages back to the console.
// DisableProxyChannel does not disable debug, use DisableDebug().
func (pe *ProxyEngine) DisableDebugChannel() {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	useDebugChannel = false
}

// EnableDebug enables printing of verbose messages during operation
func (pe *ProxyEngine) EnableDebug() {
	pe.swampopt.debug = true
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (pe *ProxyEngine) DisableDebug() {
	pe.swampopt.debug = false
}

func simpleString(s string) *strings.Builder {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString(s)
	return buf
}

func (pe *ProxyEngine) dbgPrint(builder *strings.Builder) {
	defer discardBuffer(builder)
	if !pe.swampopt.debug {
		return
	}
	if !useDebugChannel {
		pe.Debug.Print(builder.String())
		return
	}
	select {
	case debugChan <- builder.String():
		return
	default:
		buf := copABuffer.Get().(*strings.Builder)
		buf.WriteString("overflow: ")
		buf.WriteString(builder.String())
		pe.Debug.Print(buf.String())
		discardBuffer(buf)
	}
}

func (pe *ProxyEngine) msgUnableToReach(socksString string) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("unable to reach [redacted] with ")
	buf.WriteString(socksString)
	buf.WriteString(", cycling...")
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgUsingProxy(socksString string) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("MysteryDialer using socks: ")
	buf.WriteString(socksString)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgFailedMiddleware(socksString string) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("failed middleware check, ")
	buf.WriteString(socksString)
	buf.WriteString(", cycling...")
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgTry(socksString string) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("try dial with: ")
	buf.WriteString(socksString)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgCantGetLock(socksString string, putback bool) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("can't get lock for ")
	buf.WriteString(socksString)
	if putback {
		buf.WriteString(", putting back in queue")
	}
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgGotLock(socksString string) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("got lock for ")
	buf.WriteString(socksString)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgChecked(sock *Proxy, success bool) {
	buf := copABuffer.Get().(*strings.Builder)
	if success {
		buf.WriteString("verified ")
		buf.WriteString(sock.Endpoint)
		buf.WriteString(" as SOCKS")
		buf.WriteString(getProtoStr(sock.proto))
		pe.dbgPrint(buf)
		return
	}
	buf.WriteString("failed to verify: ")
	buf.WriteString(sock.Endpoint)
	pe.dbgPrint(buf)
}

func (pe *ProxyEngine) msgBadProxRate(sock *Proxy) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("badProx ratelimited: ")
	buf.WriteString(sock.Endpoint)
	pe.dbgPrint(buf)
}
