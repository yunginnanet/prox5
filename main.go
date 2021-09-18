package pxndscvm

import "errors"

const (
	grn = "\033[32m"
	ylw = "\033[33m"
	rst = "\033[0m"
)

// Start starts our proxy pool operations. Trying to start a running Swamp is a nonop.
func (s *Swamp) Start() error {
	if len(s.scvm) < 1 {
		return errors.New("there are no proxies in the list")
	}
	if !s.started {
		go s.tossUp()
		go s.feed()
		s.started = true
		return nil
	}
	return errors.New("already running")
}

/*
Pause will cease the creation of any new proxy validation operations.
   * You will be able to start the proxy pool again with Swamp.Resume(), it will have the same Statistics, options, and ratelimits.
   * During pause you are still able to dispense proxies.
   * Options may be changed and proxy lists may be loaded when paused.
   * Pausing an already paused Swamp is a nonop.
*/
func (s *Swamp) Pause() {
	if s.Status == Paused {
		return
	}
	for n := 2; n > 0; n-- {
		s.quit <- true
	}
	s.Status = Paused
}

// Resume will resume pause proxy pool operations, attempting to resume a running Swamp is a non-op.
func (s *Swamp) Resume() error {
	if s.Status == Running {
		return nil
	}
	if len(s.scvm) < 1 {
		return errors.New("there are no proxies in the list")
	}
	s.Status = Running
	go s.feed()
	go s.tossUp()
	return nil
}
