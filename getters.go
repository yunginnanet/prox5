package prox5

import (
	"strconv"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// GetStatistics returns all current statistics.
// * This is a pointer, do not modify it!
func (p5 *Swamp) GetStatistics() *statistics {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.stats
}

// RandomUserAgent retrieves a random user agent from our list in string form.
func (p5 *Swamp) RandomUserAgent() string {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return entropy.RandomStrChoice(p5.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our Swamp's options
func (p5 *Swamp) GetRandomEndpoint() string {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return entropy.RandomStrChoice(p5.swampopt.checkEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (p5 *Swamp) GetStaleTime() time.Duration {
	p5.swampopt.RLock()
	defer p5.swampopt.RUnlock()
	return p5.swampopt.stale
}

// GetValidationTimeout returns the current value of validationTimeout.
func (p5 *Swamp) GetValidationTimeout() time.Duration {
	p5.swampopt.RLock()
	defer p5.swampopt.RUnlock()
	return p5.swampopt.validationTimeout
}

// GetValidationTimeoutStr returns the current value of validationTimeout (in seconds string).
func (p5 *Swamp) GetValidationTimeoutStr() string {
	p5.swampopt.RLock()
	defer p5.swampopt.RUnlock()
	timeout := p5.swampopt.validationTimeout
	return strconv.Itoa(int(timeout / time.Second))
}

// GetServerTimeout returns the current value of serverTimeout.
func (p5 *Swamp) GetServerTimeout() time.Duration {
	p5.swampopt.RLock()
	defer p5.swampopt.RUnlock()
	return p5.swampopt.serverTimeout
}

// GetServerTimeoutStr returns the current value of serverTimeout (in seconds string).
func (p5 *Swamp) GetServerTimeoutStr() string {
	p5.swampopt.RLock()
	defer p5.swampopt.RUnlock()
	timeout := p5.swampopt.serverTimeout
	if timeout == time.Duration(0) {
		return "-1"
	}
	return strconv.Itoa(int(timeout / time.Second))
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (p5 *Swamp) GetMaxWorkers() int {
	return p5.pool.Cap()
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (p5 *Swamp) IsRunning() bool {
	return atomic.LoadUint32(&p5.Status) == 0
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (p5 *Swamp) GetRecyclingStatus() bool {
	p5.swampopt.RLock()
	defer p5.swampopt.RUnlock()
	return p5.swampopt.recycle
}

// GetWorkers retrieves pond worker statistics:
//   - return MaxWorkers, RunningWorkers, IdleWorkers
func (p5 *Swamp) GetWorkers() (maxWorkers, runningWorkers, idleWorkers int) {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.pool.Cap(), p5.pool.Running(), p5.pool.Free()
}

// GetRemoveAfter retrieves the removeafter policy, the amount of times a recycled proxy is marked as bad until it is removed entirely.
//   - returns -1 if recycling is disabled.
func (p5 *Swamp) GetRemoveAfter() int {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	if !p5.swampopt.recycle {
		return -1
	}
	return p5.swampopt.removeafter
}

// GetDialerBailout retrieves the dialer bailout policy. See SetDialerBailout for more info.
func (p5 *Swamp) GetDialerBailout() int {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.swampopt.dialerBailout
}

// TODO: Document middleware concept

func (p5 *Swamp) GetDispenseMiddleware() func(*Proxy) (*Proxy, bool) {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.dispenseMiddleware
}

func (p5 *Swamp) GetShuffleStatus() bool {
	p5.mu.RLock()
	defer p5.mu.RUnlock()
	return p5.swampopt.shuffle
}
