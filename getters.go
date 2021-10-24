package Prox5

import (
	"strconv"
	"time"
)

// GetProto safely retrieves the protocol value of the Proxy.
func (sock *Proxy) GetProto() string {
	return sock.Proto.Load().(string)
}

// RandomUserAgent retrieves a random user agent from our list in string form.
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our Swamp's options
func (s *Swamp) GetRandomEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.checkEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (s *Swamp) GetStaleTime() time.Duration {
	return s.swampopt.stale.Load().(time.Duration)
}

// GetValidationTimeout returns the current value of validationTimeout.
func (s *Swamp) GetValidationTimeout() time.Duration {
	return s.swampopt.validationTimeout.Load().(time.Duration)
}

// GetTimeoutSecondsStr returns the current value of validationTimeout (in seconds string).
func (s *Swamp) GetTimeoutSecondsStr() string {
	timeout := s.swampopt.validationTimeout.Load().(time.Duration)
	return strconv.Itoa(int(timeout / time.Second))
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (s *Swamp) GetMaxWorkers() int {
	return s.pool.Cap()
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (s *Swamp) IsRunning() bool {
	if s.runningdaemons.Load() == nil {
		println("nil")
		return false
	}
	return s.runningdaemons.Load().(int) > 0
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (s *Swamp) GetRecyclingStatus() bool {
	return s.swampopt.recycle.Load().(bool)
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
	if !s.swampopt.recycle.Load().(bool) {
		return -1
	}
	return s.swampopt.removeafter.Load().(int)
}

// GetDialerBailout retrieves the dialer bailout policy. See SetDialerBailout for more info.
func (s *Swamp) GetDialerBailout() int {
	return s.swampopt.dialerBailout.Load().(int)
}
