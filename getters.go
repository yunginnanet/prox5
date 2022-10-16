package prox5

import (
	"strconv"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// GetStatistics returns all current Statistics.
// * This is a pointer, do not modify it!
func (p5 *ProxyEngine) GetStatistics() *Statistics {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.stats
}

// RandomUserAgent retrieves a random user agent from our list in string form.
func (p5 *ProxyEngine) RandomUserAgent() string {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return entropy.RandomStrChoice(p5.opt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our ProxyEngine's options
func (p5 *ProxyEngine) GetRandomEndpoint() string {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return entropy.RandomStrChoice(p5.opt.checkEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (p5 *ProxyEngine) GetStaleTime() time.Duration {
	p5.opt.RLock()
	defer p5.opt.RUnlock()
	return p5.opt.stale
}

// GetValidationTimeout returns the current value of validationTimeout.
func (p5 *ProxyEngine) GetValidationTimeout() time.Duration {
	p5.opt.RLock()
	defer p5.opt.RUnlock()
	return p5.opt.validationTimeout
}

// GetValidationTimeoutStr returns the current value of validationTimeout (in seconds string).
func (p5 *ProxyEngine) GetValidationTimeoutStr() string {
	p5.opt.RLock()
	defer p5.opt.RUnlock()
	timeout := p5.opt.validationTimeout
	return strconv.Itoa(int(timeout / time.Second))
}

// GetServerTimeout returns the current value of serverTimeout.
func (p5 *ProxyEngine) GetServerTimeout() time.Duration {
	p5.opt.RLock()
	defer p5.opt.RUnlock()
	return p5.opt.serverTimeout
}

// GetServerTimeoutStr returns the current value of serverTimeout (in seconds string).
func (p5 *ProxyEngine) GetServerTimeoutStr() string {
	p5.opt.RLock()
	defer p5.opt.RUnlock()
	timeout := p5.opt.serverTimeout
	if timeout == time.Duration(0) {
		return "-1"
	}
	return strconv.Itoa(int(timeout / time.Second))
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (p5 *ProxyEngine) GetMaxWorkers() int {
	return p5.pool.Cap()
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (p5 *ProxyEngine) IsRunning() bool {
	return atomic.LoadUint32(&p5.Status) == 0
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (p5 *ProxyEngine) GetRecyclingStatus() bool {
	p5.opt.RLock()
	defer p5.opt.RUnlock()
	return p5.opt.recycle
}

// GetWorkers retrieves pond worker Statistics:
//   - return MaxWorkers, RunningWorkers, IdleWorkers
func (p5 *ProxyEngine) GetWorkers() (maxWorkers, runningWorkers, idleWorkers int) {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.pool.Cap(), p5.pool.Running(), p5.pool.Free()
}

// GetRemoveAfter retrieves the removeafter policy, the amount of times a recycled proxy is marked as bad until it is removed entirely.
//   - returns -1 if recycling is disabled.
func (p5 *ProxyEngine) GetRemoveAfter() int {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	if !p5.opt.recycle {
		return -1
	}
	return p5.opt.removeafter
}

// GetDialerBailout retrieves the dialer bailout policy. See SetDialerBailout for more info.
func (p5 *ProxyEngine) GetDialerBailout() int {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.opt.dialerBailout
}

// TODO: Document middleware concept

func (p5 *ProxyEngine) GetDispenseMiddleware() func(*Proxy) (*Proxy, bool) {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.dispenseMiddleware
}

// TODO: List shuffling

/*func (p5 *ProxyEngine) GetShuffleStatus() bool {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.opt.shuffle
}*/

func (p5 *ProxyEngine) GetAutoScalerStatus() bool {
	return p5.scaler.IsOn()
}

func (p5 *ProxyEngine) GetAutoScalerStateString() string {
	return p5.scaler.StateString()
}

func (p5 *ProxyEngine) GetDebugRedactStatus() bool {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.opt.redact
}
