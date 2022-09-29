package prox5

import (
	"time"

	"git.tcp.direct/kayos/prox5/logger"
)

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (p5 *Swamp) AddUserAgents(uagents []string) {
	p5.mu.Lock()
	p5.swampopt.userAgents = append(p5.swampopt.userAgents, uagents...)
	p5.mu.Unlock()
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (p5 *Swamp) SetUserAgents(uagents []string) {
	p5.mu.Lock()
	p5.swampopt.userAgents = uagents
	p5.mu.Unlock()
}

// SetCheckEndpoints replaces the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (p5 *Swamp) SetCheckEndpoints(newendpoints []string) {
	p5.mu.Lock()
	p5.swampopt.checkEndpoints = newendpoints
	p5.mu.Unlock()
}

// AddCheckEndpoints appends entries to the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (p5 *Swamp) AddCheckEndpoints(endpoints []string) {
	p5.mu.Lock()
	p5.swampopt.checkEndpoints = append(p5.swampopt.checkEndpoints, endpoints...)
	p5.mu.Unlock()
}

// SetStaleTime replaces the duration of time after which a proxy will be considered "stale". stale proxies will be skipped upon retrieval.
func (p5 *Swamp) SetStaleTime(newtime time.Duration) {
	p5.swampopt.Lock()
	p5.swampopt.stale = newtime
	p5.swampopt.Unlock()
}

// SetValidationTimeout sets the validationTimeout option.
func (p5 *Swamp) SetValidationTimeout(timeout time.Duration) {
	p5.swampopt.Lock()
	p5.swampopt.validationTimeout = timeout
	p5.swampopt.Unlock()
}

// SetServerTimeout sets the serverTimeout option.
// * serverTimeout defines the timeout for outgoing connections made with the MysteryDialer.
// * To disable timeout on outgoing MysteryDialer connections, set this to time.Duration(0).
func (p5 *Swamp) SetServerTimeout(timeout time.Duration) {
	p5.swampopt.Lock()
	p5.swampopt.serverTimeout = timeout
	p5.swampopt.Unlock()
}

// SetMaxWorkers set the maximum workers for proxy checking and clears the current proxy map and worker pool jobs.
func (p5 *Swamp) SetMaxWorkers(num int) {
	p5.pool.Tune(num)
}

// EnableRecycling enables recycling used proxies back into the pending channel for revalidation after dispensed.
func (p5 *Swamp) EnableRecycling() {
	p5.swampopt.Lock()
	p5.swampopt.recycle = true
	p5.swampopt.Unlock()
}

// DisableRecycling disables recycling used proxies back into the pending channel for revalidation after dispensed.
func (p5 *Swamp) DisableRecycling() {
	p5.swampopt.Lock()
	p5.swampopt.recycle = false
	p5.swampopt.Unlock()
}

// SetRemoveAfter sets the removeafter policy, the amount of times a recycled proxy is marked as bad before it is removed entirely.
//   - Default is 10
//   - To disable deleting entirely, set this value to -1
//   - Only applies when recycling is enabled
func (p5 *Swamp) SetRemoveAfter(timesfailed int) {
	p5.swampopt.Lock()
	p5.swampopt.removeafter = timesfailed
	p5.swampopt.Unlock()
}

// SetDialerBailout sets the amount of times the MysteryDialer will dial out and fail before it bails out.
//   - The dialer will attempt to redial a destination with a different proxy a specified amount of times before it gives up
func (p5 *Swamp) SetDialerBailout(dialattempts int) {
	p5.swampopt.Lock()
	p5.swampopt.dialerBailout = dialattempts
	p5.swampopt.Unlock()
}

// SetDispenseMiddleware will add a function that sits within the dialing process of the MysteryDialer and anyhing using it.
// This means this function will be called mid-dial during connections. Return true to approve proxy, false to skip it.
// Take care modiying the proxy in-flight as it is a pointer.
func (p5 *Swamp) SetDispenseMiddleware(f func(*Proxy) (*Proxy, bool)) {
	p5.mu.Lock()
	p5.dispenseMiddleware = f
	p5.mu.Unlock()
}

// SetDebugLogger sets the debug logger for the Swamp. See the Logger interface for implementation details.
func (p5 *Swamp) SetDebugLogger(l logger.Logger) {
	debugHardLock.Lock()
	p5.mu.Lock()
	p5.DebugLogger = l
	p5.mu.Unlock()
	debugHardLock.Unlock()
}

func (p5 *Swamp) SetShuffle(shuffle bool) {
	p5.swampopt.Lock()
	p5.swampopt.shuffle = shuffle
	p5.swampopt.Unlock()
}
