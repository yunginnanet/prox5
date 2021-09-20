package pxndscvm

import "errors"

// Start starts our proxy pool operations. Trying to start a running Swamp is a nonop.
func (s *Swamp) Start() error {
	if s.started {
		return errors.New("already running")
	}
	// mapBuilder builds deduplicated map with valid ips and ports
	go s.mapBuilder()
	// tossUp feeds jobs to pond continuously
	go s.jobSpawner()
	s.started = true
	s.mu.Lock()
	s.Status = Running
	s.mu.Unlock()
	return nil
}

/*
Pause will cease the creation of any new proxy validation operations.
   * You will be able to start the proxy pool again with Swamp.Resume(), it will have the same Statistics, options, and ratelimits.
   * During pause you are still able to dispense proxies.
   * Options may be changed and proxy lists may be loaded when paused.
   * Pausing an already paused Swamp is a nonop.
*/
func (s *Swamp) Pause() error {
	if s.Status == Paused {
		return errors.New("already paused")
	}
	for n := 2; n > 0; n-- {
		s.quit <- true
	}
	s.Status = Paused
	return nil
}

// Resume will resume pause proxy pool operations, attempting to resume a running Swamp is a non-op.
func (s *Swamp) Resume() error {
	if s.Status != Paused {
		return errors.New("not paused")
	}
	s.Status = Running
	go s.mapBuilder()
	go s.jobSpawner()
	return nil
}
