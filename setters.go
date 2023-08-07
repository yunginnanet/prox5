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
	p5.DebugLogger.Printf("added %d useragents to cycle through for proxy validation", len(uagents))
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (p5 *ProxyEngine) SetUserAgents(uagents []string) {
	p5.mu.Lock()
	p5.opt.userAgents = uagents
	p5.mu.Unlock()
	p5.DebugLogger.Printf("set %d useragents to cycle through for proxy validation", len(uagents))
}

// SetCheckEndpoints replaces the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (p5 *ProxyEngine) SetCheckEndpoints(newendpoints []string) {
	p5.mu.Lock()
	p5.opt.checkEndpoints = newendpoints
	p5.mu.Unlock()
	p5.DebugLogger.Printf("set %d check endpoints for proxy validations", len(newendpoints))
}

// AddCheckEndpoints appends entries to the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (p5 *ProxyEngine) AddCheckEndpoints(endpoints []string) {
	p5.mu.Lock()
	p5.opt.checkEndpoints = append(p5.opt.checkEndpoints, endpoints...)
	p5.mu.Unlock()
	p5.DebugLogger.Printf("added %d check endpoints for proxy validations", len(endpoints))
}

// SetStaleTime replaces the duration of time after which a proxy will be considered "stale". stale proxies will be skipped upon retrieval.
func (p5 *ProxyEngine) SetStaleTime(newtime time.Duration) {
	p5.opt.Lock()
	p5.opt.stale = newtime
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 stale time set to %s", newtime)
}

// SetValidationTimeout sets the validationTimeout option.
func (p5 *ProxyEngine) SetValidationTimeout(timeout time.Duration) {
	p5.opt.Lock()
	p5.opt.validationTimeout = timeout
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 validation timeout set to %s", timeout)
}

// SetServerTimeout sets the serverTimeout option.
// * serverTimeout defines the timeout for outgoing connections made with the mysteryDialer.
// * To disable timeout on outgoing mysteryDialer connections, set this to time.Duration(0).
func (p5 *ProxyEngine) SetServerTimeout(timeout time.Duration) {
	p5.opt.Lock()
	p5.opt.serverTimeout = timeout
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 server timeout set to %s", timeout)
}

// SetMaxWorkers set the maximum workers for proxy checking.
func (p5 *ProxyEngine) SetMaxWorkers(num int) {
	if p5.isEmpty() && num < 2 {
		p5.DebugLogger.
			Printf("prox5 cannot set max workers to %d, minimum is 2 until we have some valid proxies", num)
		num = 2
	}
	p5.pool.Tune(num)
	p5.scaler.SetBaseline(num)

}

// EnableRecycling enables recycling used proxies back into the pending channel for revalidation after dispensed.
func (p5 *ProxyEngine) EnableRecycling() {
	p5.opt.Lock()
	p5.opt.recycle = true
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 recycling enabled")
}

// DisableRecycling disables recycling used proxies back into the pending channel for revalidation after dispensed.
func (p5 *ProxyEngine) DisableRecycling() {
	p5.opt.Lock()
	p5.opt.recycle = false
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 recycling disabled")
}

// SetRemoveAfter sets the removeafter policy, the amount of times a recycled proxy is marked as bad before it is removed entirely.
//   - Default is 10
//   - To disable deleting entirely, set this value to -1
//   - Only applies when recycling is enabled
func (p5 *ProxyEngine) SetRemoveAfter(timesfailed int) {
	p5.opt.Lock()
	p5.opt.removeafter = timesfailed
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 removeafter policy set to %d", timesfailed)
}

// SetDialerBailout sets the amount of times the mysteryDialer will dial out and fail before it bails out.
//   - The dialer will attempt to redial a destination with a different proxy a specified amount of times before it gives up
func (p5 *ProxyEngine) SetDialerBailout(dialattempts int) {
	p5.opt.Lock()
	p5.opt.dialerBailout = dialattempts
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 dialer bailout set to %d", dialattempts)
}

// SetDispenseMiddleware will add a function that sits within the dialing process of the mysteryDialer and anyhing using it.
// This means this function will be called mid-dial during connections. Return true to approve proxy, false to skip it.
// Take care modiying the proxy in-flight as it is a pointer.
func (p5 *ProxyEngine) SetDispenseMiddleware(f func(*Proxy) (*Proxy, bool)) {
	p5.mu.Lock()
	p5.dispenseMiddleware = f
	p5.mu.Unlock()
	p5.DebugLogger.Printf("prox5 dispense middleware set")
}

// SetDebugLogger sets the debug logger for the ProxyEngine. See the Logger interface for implementation details.
//
// Deprecated: use SetLogger instead. This will be removed in a future version.
func (p5 *ProxyEngine) SetDebugLogger(l logger.Logger) {
	p5.SetLogger(l)
}

// SetLogger sets the debug logger for the ProxyEngine. See the Logger interface for implementation details.
func (p5 *ProxyEngine) SetLogger(l logger.Logger) {
	debugHardLock.Lock()
	p5.mu.Lock()
	p5.DebugLogger = l
	p5.mu.Unlock()
	debugHardLock.Unlock()
	p5.DebugLogger.Printf("prox5 debug logger set")
}

func (p5 *ProxyEngine) SetAndEnableDebugLogger(l logger.Logger) {
	p5.SetLogger(l)
	p5.EnableDebug()
}

// EnableAutoScaler enables the autoscaler.
// This will automatically scale up the number of workers based on the threshold of dial attempts versus validated proxies.
func (p5 *ProxyEngine) EnableAutoScaler() {
	p5.scaler.Enable()
	p5.DebugLogger.Printf("prox5 autoscaler enabled")
}

// DisableAutoScaler disables the autoscaler.
func (p5 *ProxyEngine) DisableAutoScaler() {
	p5.scaler.Disable()
	p5.DebugLogger.Printf("prox5 autoscaler disabled")
}

// SetAutoScalerMaxScale sets the relative maximum amount that the autoscaler will scale up.
func (p5 *ProxyEngine) SetAutoScalerMaxScale(max int) {
	p5.scaler.SetMax(max)
	p5.DebugLogger.Printf("prox5 autoscaler max scale set to %d", max)
}

// SetAutoScalerThreshold sets the threshold of validated proxies versus dials that will trigger the autoscaler.
func (p5 *ProxyEngine) SetAutoScalerThreshold(threshold int) {
	p5.scaler.SetThreshold(threshold)
	p5.DebugLogger.Printf("prox5 autoscaler threshold set to %d", threshold)
}

func (p5 *ProxyEngine) EnableDebugRedaction() {
	p5.opt.Lock()
	p5.opt.redact = true
	p5.opt.Unlock()
	p5.DebugLogger.Printf("[redacted]")
}

func (p5 *ProxyEngine) DisableDebugRedaction() {
	p5.opt.Lock()
	p5.opt.redact = false
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 redaction disabled")
}

func (p5 *ProxyEngine) EnableRecyclerShuffling() {
	p5.opt.Lock()
	p5.opt.shuffle = true
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 recycler shuffling enabled")
}

func (p5 *ProxyEngine) DisableRecyclerShuffling() {
	p5.opt.Lock()
	p5.opt.shuffle = false
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 recycler shuffling disabled")
}

func (p5 *ProxyEngine) EnableHTTPClientTLSVerification() {
	p5.opt.Lock()
	p5.opt.tlsVerify = true
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 HTTP client TLS verification enabled")
}

func (p5 *ProxyEngine) DisableHTTPClientTLSVerification() {
	p5.opt.Lock()
	p5.opt.tlsVerify = false
	p5.opt.Unlock()
	p5.DebugLogger.Printf("prox5 HTTP client TLS verification disabled")
}
