package pxndscvm

import (
	"errors"
	"fmt"
	"time"

	"github.com/alitto/pond"
)

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (s *Swamp) AddUserAgents(uagents []string) {
	// mutex lock so that RLock during proxy checking will block while we change this value
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.userAgents = append(s.swampopt.userAgents, uagents...)
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (s *Swamp) SetUserAgents(uagents []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.userAgents = append(s.swampopt.userAgents, uagents...)
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
	s.swampopt.stale = newtime
}

// SetValidationTimeout sets the validationTimeout option (in seconds).
func (s *Swamp) SetValidationTimeout(newtimeout int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.validationTimeout = newtimeout
}

// SetMaxWorkers set the maximum workers for proxy checking, this must be set before calling LoadProxyTXT for the first time.
func (s *Swamp) SetMaxWorkers(num int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Status == Running {
		return errors.New("can't change max workers during proxypool operation, try pausing first")
	}
	s.pool.StopAndWait()
	s.swampopt.maxWorkers = num
	s.pool = pond.New(s.swampopt.maxWorkers, 1000000, pond.PanicHandler(func(p interface{}) {
		fmt.Println("WORKER PANIC! ", p)
	}))
	return nil
}

// EnableRecycling toggles recycling used proxies back into the pending channel for revalidation after dispensed.
func (s *Swamp) EnableRecycling(choice bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.recycle = choice
}
