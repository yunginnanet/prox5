package prox5

import (
	"sync"
)

var (
	useDebugChannel = false
	debugChan       chan string
	debugMutex      *sync.RWMutex
)

func init() {
	debugMutex = &sync.RWMutex{}
}

const (
	grn = "\033[32m"
	red = "\033[31m"
	ylw = "\033[33m"
	rst = "\033[0m"
)

// DebugChannel will return a channel which will receive debug messages once debug is enabled.
// This will alter the flow of debug messages, they will no longer print to console, they will be pushed into this channel.
// Make sure you pull from the channel eventually to avoid build up of blocked goroutines.
//
// Note that this will replace any existing debug channel with a fresh one.
func (s *Swamp) DebugChannel() chan string {
	debugChan = make(chan string, 2048)
	useDebugChannel = true
	return debugChan
}

// IsDebugEnabled returns the current state of our debug switch.
func (s *Swamp) IsDebugEnabled() bool {
	debugMutex.RLock()
	defer debugMutex.RUnlock()
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
	debugMutex.RLock()
	if s.swampopt.debug == false {
		return
	}
	debugMutex.RUnlock()

	if useDebugChannel {
		select {
		case debugChan <- str:
			return
		default:
			println("Prox5 overflow: " + str)
			return
		}
	}
	println("Prox5: " + str)
}
