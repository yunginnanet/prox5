package prox5

import (
	"time"
)

// AddUserAgents appends to the list of useragents we randomly choose from during proxied requests
func (s *Swamp) AddUserAgents(uagents []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.userAgents = append(s.swampopt.userAgents, uagents...)
}

// SetUserAgents sets the list of useragents we randomly choose from during proxied requests
func (s *Swamp) SetUserAgents(uagents []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.userAgents = uagents
}

// SetCheckEndpoints replaces the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (s *Swamp) SetCheckEndpoints(newendpoints []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.checkEndpoints = newendpoints
}

// AddCheckEndpoints appends entries to the running list of whatismyip style endpoitns for validation. (must return only the WAN IP)
func (s *Swamp) AddCheckEndpoints(endpoints []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.swampopt.checkEndpoints = append(s.swampopt.checkEndpoints, endpoints...)
}

// SetStaleTime replaces the duration of time after which a proxy will be considered "stale". stale proxies will be skipped upon retrieval.
func (s *Swamp) SetStaleTime(newtime time.Duration) {
	s.swampopt.stale.Store(newtime)
}

// SetValidationTimeout sets the validationTimeout option.
func (s *Swamp) SetValidationTimeout(timeout time.Duration) {
	s.swampopt.validationTimeout.Store(timeout)
}

// SetServerTimeout sets the serverTimeout option.
// * serverTimeout defines the timeout for outgoing connections made with the MysteryDialer.
// * To disable timeout on outgoing MysteryDialer connections, set this to time.Duration(0).
func (s *Swamp) SetServerTimeout(timeout time.Duration) {
	s.swampopt.serverTimeout.Store(timeout)
}

// SetMaxWorkers set the maximum workers for proxy checking and clears the current proxy map and worker pool jobs.
func (s *Swamp) SetMaxWorkers(num int) {
	s.pool.Tune(num)
}

// EnableRecycling enables recycling used proxies back into the pending channel for revalidation after dispensed.
func (s *Swamp) EnableRecycling() {
	s.swampopt.recycle.Store(true)
}

// DisableRecycling disables recycling used proxies back into the pending channel for revalidation after dispensed.
func (s *Swamp) DisableRecycling() {
	s.swampopt.recycle.Store(false)
}

// SetRemoveAfter sets the removeafter policy, the amount of times a recycled proxy is marked as bad before it is removed entirely.
//    * Default is 5
//    * To disable deleting entirely, set this value to -1
//    * Only applies when recycling is enabled
func (s *Swamp) SetRemoveAfter(timesfailed int) {
	s.swampopt.removeafter.Store(timesfailed)
}

// SetDialerBailout sets the amount of times the MysteryDialer will dial out and fail before it bails out.
//	  * The dialer will attempt to redial a destination with a different proxy a specified amount of times before it gives up
func (s *Swamp) SetDialerBailout(dialattempts int) {
	s.swampopt.dialerBailout.Store(dialattempts)
}
