package pxndscvm

import (
	"errors"
	"time"
)

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (s *Swamp) AddUserAgents(uagents []string) {
	// mutex lock so that RLock during proxy checking will block while we change this value
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.UserAgents = append(s.swampopt.UserAgents, uagents...)
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (s *Swamp) SetUserAgents(uagents []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.UserAgents = append(s.swampopt.UserAgents, uagents...)
}

// SetCheckEndpoints replaces the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (s *Swamp) SetCheckEndpoints(newendpoints []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.CheckEndpoints = newendpoints
}

// AddCheckEndpoints appends entries to the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (s *Swamp) AddCheckEndpoints(newendpoints []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.CheckEndpoints = append(s.swampopt.CheckEndpoints, newendpoints...)
}

// SetStaleTime replaces the duration of time after which a proxy will be considered "stale". stale proxies will be skipped upon retrieval.
func (s *Swamp) SetStaleTime(newtime time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.Stale = newtime
}

// SetValidationTimeout sets the ValidationTimeout option (in seconds).
func (s *Swamp) SetValidationTimeout(newtimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.ValidationTimeout = newtimeout
}

// SetMaxWorkers set the maximum workers for proxy checking, this must be set before calling LoadProxyTXT for the first time.
func (s *Swamp) SetMaxWorkers(num int) error {
	if s.Status == Running {
		return errors.New("can't change max workers during proxypool operation, try pausing first")
	}
	s.swampopt.MaxWorkers = num
	return nil
}
