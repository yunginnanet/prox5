package prox5

import (
	"time"

	"git.tcp.direct/kayos/prox5/logger"
)

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (pe *ProxyEngine) AddUserAgents(uagents []string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.swampopt.userAgents = append(pe.swampopt.userAgents, uagents...)
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (pe *ProxyEngine) SetUserAgents(uagents []string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.swampopt.userAgents = uagents
}

// SetCheckEndpoints replaces the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (pe *ProxyEngine) SetCheckEndpoints(newendpoints []string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.swampopt.checkEndpoints = newendpoints
}

// AddCheckEndpoints appends entries to the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (pe *ProxyEngine) AddCheckEndpoints(endpoints []string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.swampopt.checkEndpoints = append(pe.swampopt.checkEndpoints, endpoints...)
}

// SetStaleTime replaces the duration of time after which a proxy will be considered "stale". stale proxies will be skipped upon retrieval.
func (pe *ProxyEngine) SetStaleTime(newtime time.Duration) {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.stale = newtime
}

// SetValidationTimeout sets the validationTimeout option.
func (pe *ProxyEngine) SetValidationTimeout(timeout time.Duration) {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.validationTimeout = timeout
}

// SetServerTimeout sets the serverTimeout option.
// * serverTimeout defines the timeout for outgoing connections made with the MysteryDialer.
// * To disable timeout on outgoing MysteryDialer connections, set this to time.Duration(0).
func (pe *ProxyEngine) SetServerTimeout(timeout time.Duration) {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.serverTimeout = timeout
}

// SetMaxWorkers set the maximum workers for proxy checking and clears the current proxy map and worker pool jobs.
func (pe *ProxyEngine) SetMaxWorkers(num int) {
	pe.pool.Tune(num)
}

// EnableRecycling enables recycling used proxies back into the pending channel for revalidation after dispensed.
func (pe *ProxyEngine) EnableRecycling() {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.recycle = true
}

// DisableRecycling disables recycling used proxies back into the pending channel for revalidation after dispensed.
func (pe *ProxyEngine) DisableRecycling() {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.recycle = false
}

// SetRemoveAfter sets the removeafter policy, the amount of times a recycled proxy is marked as bad before it is removed entirely.
//   - Default is 10
//   - To disable deleting entirely, set this value to -1
//   - Only applies when recycling is enabled
func (pe *ProxyEngine) SetRemoveAfter(timesfailed int) {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.removeafter = timesfailed
}

// SetDialerBailout sets the amount of times the MysteryDialer will dial out and fail before it bails out.
//   - The dialer will attempt to redial a destination with a different proxy a specified amount of times before it gives up
func (pe *ProxyEngine) SetDialerBailout(dialattempts int) {
	pe.swampopt.Lock()
	defer pe.swampopt.Unlock()
	pe.swampopt.dialerBailout = dialattempts
}

// SetDispenseMiddleware will add a function that sits within the dialing process of the MysteryDialer and anyhing using it.
// This means this function will be called mid-dial during connections. Return true to approve proxy, false to skip it.
// Take care modiying the proxy in-flight as it is a pointer.
func (pe *ProxyEngine) SetDispenseMiddleware(f func(*Proxy) (*Proxy, bool)) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.dispenseMiddleware = f
}

// SetDebugLogger sets the debug logger for the ProxyEngine. See the Logger interface for implementation details.
func (pe *ProxyEngine) SetDebugLogger(l logger.Logger) {
	debugHardLock.Lock()
	pe.mu.Lock()
	pe.DebugLogger = l
	pe.mu.Unlock()
	debugHardLock.Unlock()
}

func (pe *ProxyEngine) SetShuffle(shuffle bool) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.swampopt.shuffle = shuffle
}
