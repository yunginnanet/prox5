package pxndscvm

import (
	"time"
)

// RandomUserAgent retrieves a random user agent from our list in string form
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our Swamp's options
func (s *Swamp) GetRandomEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.CheckEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (s *Swamp) GetStaleTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.stale
}

// GetValidationTimeout returns the current value of validationTimeout (in seconds).
func (s *Swamp) GetValidationTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.validationTimeout
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (s *Swamp) GetMaxWorkers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.maxWorkers
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (s *Swamp) IsRunning() bool {
	if s.runningdaemons == 2 {
		return true
	}
	return false
}

// GetRecyclingStatus retrieves the current recycling status, see EnableRecycling.
func (s *Swamp) GetRecyclingStatus() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.recycle
}

// GetWorkers retrieves pond worker statistics:
//    * return MaxWorkers, RunningWorkers, IdleWorkers
func (s *Swamp) GetWorkers() (int, int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return  s.pool.Cap(), s.pool.Running(), s.pool.Free()
}

// GetRemoveAfter retrieves the removeafter policy, the amount of times a recycled proxy is marked as bad until it is removed entirely.
//    *  returns -1 if recycling is disabled.
func (s *Swamp) GetRemoveAfter() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.swampopt.recycle == false {
		return -1
	}
	return s.swampopt.removeafter
}
