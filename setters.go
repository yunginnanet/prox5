package prox5

import (
	"time"

	"git.tcp.direct/kayos/prox5/logger"
)

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (p5 *ProxyEngine) AddUserAgents(uagents []string) {
	p5.mu.Lock()
	p5.opt.userAgents = append(p5.opt.userAgents, uagents...)
	p5.mu.Unlock()
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (p5 *ProxyEngine) SetUserAgents(uagents []string) {
	p5.mu.Lock()
	p5.opt.userAgents = uagents
	p5.mu.Unlock()
}

// SetCheckEndpoints replaces the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (p5 *ProxyEngine) SetCheckEndpoints(newendpoints []string) {
	p5.mu.Lock()
	p5.opt.checkEndpoints = newendpoints
	p5.mu.Unlock()
}

// AddCheckEndpoints appends entries to the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (p5 *ProxyEngine) AddCheckEndpoints(endpoints []string) {
	p5.mu.Lock()
	p5.opt.checkEndpoints = append(p5.opt.checkEndpoints, endpoints...)
	p5.mu.Unlock()
}

// SetStaleTime replaces the duration of time after which a proxy will be considered "stale". stale proxies will be skipped upon retrieval.
func (p5 *ProxyEngine) SetStaleTime(newtime time.Duration) {
	p5.opt.Lock()
	p5.opt.stale = newtime
	p5.opt.Unlock()
}

// SetValidationTimeout sets the validationTimeout option.
func (p5 *ProxyEngine) SetValidationTimeout(timeout time.Duration) {
	p5.opt.Lock()
	p5.opt.validationTimeout = timeout
	p5.opt.Unlock()
}

// SetServerTimeout sets the serverTimeout option.
// * serverTimeout defines the timeout for outgoing connections made with the MysteryDialer.
// * To disable timeout on outgoing MysteryDialer connections, set this to time.Duration(0).
func (p5 *ProxyEngine) SetServerTimeout(timeout time.Duration) {
	p5.opt.Lock()
	p5.opt.serverTimeout = timeout
	p5.opt.Unlock()
}

// SetMaxWorkers set the maximum workers for proxy checking.
func (p5 *ProxyEngine) SetMaxWorkers(num int) {
	p5.pool.Tune(num)
	p5.scaler.SetBaseline(num)
}

// EnableRecycling enables recycling used proxies back into the pending channel for revalidation after dispensed.
func (p5 *ProxyEngine) EnableRecycling() {
	p5.opt.Lock()
	p5.opt.recycle = true
	p5.opt.Unlock()
}

// DisableRecycling disables recycling used proxies back into the pending channel for revalidation after dispensed.
func (p5 *ProxyEngine) DisableRecycling() {
	p5.opt.Lock()
	p5.opt.recycle = false
	p5.opt.Unlock()
}

// SetRemoveAfter sets the removeafter policy, the amount of times a recycled proxy is marked as bad before it is removed entirely.
//   - Default is 10
//   - To disable deleting entirely, set this value to -1
//   - Only applies when recycling is enabled
func (p5 *ProxyEngine) SetRemoveAfter(timesfailed int) {
	p5.opt.Lock()
	p5.opt.removeafter = timesfailed
	p5.opt.Unlock()
}

// SetDialerBailout sets the amount of times the MysteryDialer will dial out and fail before it bails out.
//   - The dialer will attempt to redial a destination with a different proxy a specified amount of times before it gives up
func (p5 *ProxyEngine) SetDialerBailout(dialattempts int) {
	p5.opt.Lock()
	p5.opt.dialerBailout = dialattempts
	p5.opt.Unlock()
}

// SetDispenseMiddleware will add a function that sits within the dialing process of the MysteryDialer and anyhing using it.
// This means this function will be called mid-dial during connections. Return true to approve proxy, false to skip it.
// Take care modiying the proxy in-flight as it is a pointer.
func (p5 *ProxyEngine) SetDispenseMiddleware(f func(*Proxy) (*Proxy, bool)) {
	p5.mu.Lock()
	p5.dispenseMiddleware = f
	p5.mu.Unlock()
}

// SetDebugLogger sets the debug logger for the ProxyEngine. See the Logger interface for implementation details.
func (p5 *ProxyEngine) SetDebugLogger(l logger.Logger) {
	debugHardLock.Lock()
	p5.mu.Lock()
	p5.DebugLogger = l
	p5.mu.Unlock()
	debugHardLock.Unlock()
}

// EnableAutoScaler enables the autoscaler.
// This will automatically scale up the number of workers based on the threshold of dial attempts versus validated proxies.
func (p5 *ProxyEngine) EnableAutoScaler() {
	p5.scaler.Enable()
}

// DisableAutoScaler disables the autoscaler.
func (p5 *ProxyEngine) DisableAutoScaler() {
	p5.scaler.Disable()
}

// SetAutoScalerMaxScale sets the relative maximum amount that the autoscaler will scale up.
func (p5 *ProxyEngine) SetAutoScalerMaxScale(max int) {
	p5.scaler.SetMax(max)
}

// SetAutoScalerThreshold sets the threshold of validated proxies versus dials that will trigger the autoscaler.
func (p5 *ProxyEngine) SetAutoScalerThreshold(threshold int) {
	p5.scaler.SetThreshold(threshold)
}

func (p5 *ProxyEngine) EnableDebugRedaction() {
	p5.opt.Lock()
	p5.opt.redact = true
	p5.opt.Unlock()
}

func (p5 *ProxyEngine) DisableDebugRedaction() {
	p5.opt.Lock()
	p5.opt.redact = false
	p5.opt.Unlock()
}

/*func (p5 *ProxyEngine) EnableListShuffle() {
	p5.opt.Lock()
	p5.opt.shuffle = true
	p5.opt.Unlock()
}

func (p5 *ProxyEngine) DisableListShuffle() {
	p5.opt.Lock()
	p5.opt.shuffle = false
	p5.opt.Unlock()
}*/
