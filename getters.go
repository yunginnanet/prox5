package prox5

import (
	"strconv"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// GetProto retrieves the known protocol value of the Proxy.
func (sock *Proxy) GetProto() ProxyProtocol {
	return sock.proto
}

// GetStatistics returns all current statistics.
// * This is a pointer, do not modify it!
func (s *Swamp) GetStatistics() *statistics {
	return s.stats
}

// RandomUserAgent retrieves a random user agent from our list in string form.
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return entropy.RandomStrChoice(s.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our Swamp's options
func (s *Swamp) GetRandomEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return entropy.RandomStrChoice(s.swampopt.checkEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (s *Swamp) GetStaleTime() time.Duration {
	s.swampopt.RLock()
	defer s.swampopt.RLock()
	return s.swampopt.stale
}

// GetValidationTimeout returns the current value of validationTimeout.
func (s *Swamp) GetValidationTimeout() time.Duration {
	s.swampopt.RLock()
	defer s.swampopt.RLock()
	return s.swampopt.validationTimeout
}

// GetValidationTimeoutStr returns the current value of validationTimeout (in seconds string).
func (s *Swamp) GetValidationTimeoutStr() string {
	s.swampopt.RLock()
	defer s.swampopt.RLock()
	timeout := s.swampopt.validationTimeout
	return strconv.Itoa(int(timeout / time.Second))
}

// GetServerTimeout returns the current value of serverTimeout.
func (s *Swamp) GetServerTimeout() time.Duration {
	s.swampopt.RLock()
	defer s.swampopt.RLock()
	return s.swampopt.serverTimeout
}

// GetServerTimeoutStr returns the current value of serverTimeout (in seconds string).
func (s *Swamp) GetServerTimeoutStr() string {
	s.swampopt.RLock()
	defer s.swampopt.RLock()
	timeout := s.swampopt.serverTimeout
	if timeout == time.Duration(0) {
		return "-1"
	}
	return strconv.Itoa(int(timeout / time.Second))
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (s *Swamp) GetMaxWorkers() int {
	return s.pool.Cap()
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (s *Swamp) IsRunning() bool {
	return atomic.LoadInt32(&s.runningdaemons) > 0
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (s *Swamp) GetRecyclingStatus() bool {
	s.swampopt.RLock()
	defer s.swampopt.RLock()
	return s.swampopt.recycle
}

// GetWorkers retrieves pond worker statistics:
//    * return MaxWorkers, RunningWorkers, IdleWorkers
func (s *Swamp) GetWorkers() (maxWorkers, runningWorkers, idleWorkers int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pool.Cap(), s.pool.Running(), s.pool.Free()
}

// GetRemoveAfter retrieves the removeafter policy, the amount of times a recycled proxy is marked as bad until it is removed entirely.
//    *  returns -1 if recycling is disabled.
func (s *Swamp) GetRemoveAfter() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.swampopt.recycle {
		return -1
	}
	return s.swampopt.removeafter
}

// GetDialerBailout retrieves the dialer bailout policy. See SetDialerBailout for more info.
func (s *Swamp) GetDialerBailout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.dialerBailout
}

// TODO: More docs
func (s *Swamp) GetDispenseMiddleware() func(*Proxy) (*Proxy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dispenseMiddleware
}
