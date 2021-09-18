package pxndscvm

import (
	"sync"
	"time"
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
	debugMutex.Lock()
	defer debugMutex.Unlock()
	if !s.DebugEnabled() {
		s.EnableDebug()
	}
	debugChan = make(chan string, 1000)
	useDebugChannel = true
	return debugChan
}

// DisableDebugChannel redirects debug messages back to the console.
// DisableProxyChannel does not disable debug, use DisableDebug().
func (s *Swamp) DisableDebugChannel() chan string {
	useDebugChannel = false

	// Just in case..?
	time.Sleep(100 * time.Millisecond)

	close(debugChan)
	useDebugChannel = true
	return debugChan
}

// EnableDebug enables printing of verbose messages during operation
func (s *Swamp) EnableDebug() {
	s.mu.Lock()
	debugMutex.Lock()
	defer s.mu.Unlock()
	defer debugMutex.Unlock()
	s.swampopt.Debug = true
}

// DisableDebug enables printing of verbose messages during operation.
// WARNING: if you are using a DebugChannel, you must read all of the messages in the channel's cache or this will block.
func (s *Swamp) DisableDebug() {
	s.mu.Lock()
	debugMutex.Lock()
	defer s.mu.Unlock()
	defer debugMutex.Unlock()
	s.swampopt.Debug = false
}

func (s *Swamp) dbgPrint(str string) {
	debugMutex.RLock()
	if !s.swampopt.Debug {
		return
	}

	if useDebugChannel {
		go func() {
			defer debugMutex.RUnlock()
			debugChan <- str
		}()
		return
	}

	debugMutex.RUnlock()
	println("pxndscvm: " + str)
}
