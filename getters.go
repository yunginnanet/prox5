package prox5

import (
	"strconv"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// GetStatistics returns all current statistics.
// * This is a pointer, do not modify it!
func (pe *Swamp) GetStatistics() *statistics {
	return pe.stats
}

// RandomUserAgent retrieves a random user agent from our list in string form.
func (pe *Swamp) RandomUserAgent() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return entropy.RandomStrChoice(pe.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our Swamp's options
func (pe *Swamp) GetRandomEndpoint() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return entropy.RandomStrChoice(pe.swampopt.checkEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (pe *Swamp) GetStaleTime() time.Duration {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.stale
}

// GetValidationTimeout returns the current value of validationTimeout.
func (pe *Swamp) GetValidationTimeout() time.Duration {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.validationTimeout
}

// GetValidationTimeoutStr returns the current value of validationTimeout (in seconds string).
func (pe *Swamp) GetValidationTimeoutStr() string {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	timeout := pe.swampopt.validationTimeout
	return strconv.Itoa(int(timeout / time.Second))
}

// GetServerTimeout returns the current value of serverTimeout.
func (pe *Swamp) GetServerTimeout() time.Duration {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.serverTimeout
}

// GetServerTimeoutStr returns the current value of serverTimeout (in seconds string).
func (pe *Swamp) GetServerTimeoutStr() string {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	timeout := pe.swampopt.serverTimeout
	if timeout == time.Duration(0) {
		return "-1"
	}
	return strconv.Itoa(int(timeout / time.Second))
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (pe *Swamp) GetMaxWorkers() int {
	return pe.pool.Cap()
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (pe *Swamp) IsRunning() bool {
	return atomic.LoadInt32(&pe.runningdaemons) == 2
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (pe *Swamp) GetRecyclingStatus() bool {
	pe.swampopt.RLock()
	defer pe.swampopt.RLock()
	return pe.swampopt.recycle
}

// GetWorkers retrieves pond worker statistics:
//   - return MaxWorkers, RunningWorkers, IdleWorkers
func (pe *Swamp) GetWorkers() (maxWorkers, runningWorkers, idleWorkers int) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.pool.Cap(), pe.pool.Running(), pe.pool.Free()
}

// GetRemoveAfter retrieves the removeafter policy, the amount of times a recycled proxy is marked as bad until it is removed entirely.
//   - returns -1 if recycling is disabled.
func (pe *Swamp) GetRemoveAfter() int {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	if !pe.swampopt.recycle {
		return -1
	}
	return pe.swampopt.removeafter
}

// GetDialerBailout retrieves the dialer bailout policy. See SetDialerBailout for more info.
func (pe *Swamp) GetDialerBailout() int {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.swampopt.dialerBailout
}

// TODO: Document middleware concept

func (pe *Swamp) GetDispenseMiddleware() func(*Proxy) (*Proxy, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.dispenseMiddleware
}

func (pe *Swamp) GetShuffleStatus() bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.swampopt.shuffle
}
