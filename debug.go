package pxndscvm

import (
	"sync"

	rate5 "github.com/yunginnanet/Rate5"
)

var (
	useDebugChannel = false
	debugChan       chan string
	debugMutex      *sync.RWMutex
	debugRatelimit  *rate5.Limiter
)

type debugLine struct {
	s string
}

// UniqueKey implements rate5's Identity interface.
// https://pkg.go.dev/github.com/yunginnanet/Rate5#Identity
func (dbg debugLine) UniqueKey() string {
	return dbg.s
}

func init() {
	debugMutex = &sync.RWMutex{}
	debugRatelimit = rate5.NewStrictLimiter(120, 2)
}

// DebugChannel will return a channel which will receive debug messages once debug is enabled.
// This will alter the flow of debug messages, they will no longer print to console, they will be pushed into this channel.
// Make sure you pull from the channel eventually to avoid build up of blocked goroutines.
func (s *Swamp) DebugChannel() chan string {
	debugChan = make(chan string, 1000)
	useDebugChannel = true
	return debugChan
}

// DebugEnabled returns the current state of our debug switch.
func (s *Swamp) DebugEnabled() bool {
	return s.swampopt.debug
}

// DisableDebugChannel redirects debug messages back to the console.
// DisableProxyChannel does not disable debug, use DisableDebug().
func (s *Swamp) DisableDebugChannel() {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	useDebugChannel = false
}

// EnableDebug enables printing of verbose messages during operation
func (s *Swamp) EnableDebug() {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	s.swampopt.debug = true
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (s *Swamp) DisableDebug() {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	s.swampopt.debug = false
}

func (s *Swamp) dbgPrint(str string) {
	if !s.swampopt.debug {
		return
	}

	if debugRatelimit.Check(debugLine{s: str}) {
		return
	}

	if useDebugChannel {
		go func() {
			debugChan <- str
		}()
		return
	}
	println("pxndscvm: " + str)
}
