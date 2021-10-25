package Prox5

import "errors"

// SwampStatus represents the current state of our Swamp.
type SwampStatus uint32

const (
	// Running means the proxy pool is currently taking in proxys and validating them, and is available to dispense proxies.
	Running SwampStatus = iota
	// Paused means the proxy pool has been with Swamp.Pause() and may be resumed with Swamp.Resume()
	Paused
	// New means the proxy pool has never been started.
	New
)

// Start starts our proxy pool operations. Trying to start a running Swamp will return an error.
func (s *Swamp) Start() error {
	if s.Status.Load().(SwampStatus) != New {
		return errors.New("this swamp is not new, use resume if it is paused")
	}

	s.runningdaemons.Store(0)

	s.getThisDread()

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
	if !s.IsRunning() {
		return errors.New("not running")
	}

	s.dbgPrint("pausing...")

	s.svcDown()
	s.svcDown()

	s.Status.Store(Paused)
	return nil
}

func (s *Swamp) getThisDread() {
	go s.mapBuilder()
	<-s.conductor
	go s.jobSpawner()

	for {
		if s.IsRunning() {
			s.Status.Store(Running)
			break
		}
	}
}

// Resume will resume pause proxy pool operations, attempting to resume a running Swamp is returns an error.
func (s *Swamp) Resume() error {
	if s.IsRunning() {
		return errors.New("not paused")
	}

	s.getThisDread()

	return nil
}
