package pxndscvm

import (
	"sync"
)

var (
	useDebugChannel = false
	debugChan chan string
	debugMutex *sync.RWMutex
)

func init() {
	debugMutex = &sync.RWMutex{}
}

// DebugChannel will enable debug if it's not already enabled and return a channel which will receive debug messages.
// This will alter the flow of debug messages, they will no longer print to console, they will be pushed into this channel.
// Make sure you pull from the channel eventually to avoid build up of blocked goroutines.
func (s *Swamp) DebugChannel() chan string {
	if !s.DebugEnabled() {
		s.EnableDebug()
	}
	debugMutex.Lock()
	debugChan = make(chan string, 1000)
	useDebugChannel = true
	debugMutex.Unlock()
	return debugChan
}

// DisableDebugChannel redirects debug messages back to the console.
// DisableProxyChannel does not disable debug, use DisableDebug().
func (s *Swamp) DisableDebugChannel() chan string {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	useDebugChannel = false
	close(debugChan)
}

// EnableDebug enables printing of verbose messages during operation
func (s *Swamp) EnableDebug() {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	s.swampopt.Debug = true
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (s *Swamp) DisableDebug() {
	debugMutex.Lock()
	defer debugMutex.Unlock()
	s.swampopt.Debug = false
	close(debugChan)
}

func (s *Swamp) dbgPrint(str string) {
	debugMutex.RLock()
	defer debugMutex.RUnlock()
	if !s.swampopt.Debug {
		return
	}
	if useDebugChannel {
		go func() {
			debugChan <- str
			println("sent down channel: " + str)
		}()
		return
	}
	println("pxndscvm: " + str)
}
