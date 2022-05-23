package prox5

import (
	"errors"
	"sync/atomic"
)

// SwampStatus represents the current state of our Swamp.
type SwampStatus uint32

const (
	// StateRunning means the proxy pool is currently taking in proxys and validating them, and is available to dispense proxies.
	StateRunning SwampStatus = iota
	// StatePaused means the proxy pool has been with Swamp.Pause() and may be resumed with Swamp.Resume()
	StatePaused
	// StateNew means the proxy pool has never been started.
	StateNew
)

// Start starts our proxy pool operations. Trying to start a running Swamp will return an error.
func (s *Swamp) Start() error {
	switch {
	case s.IsRunning():
		return errors.New("pool is already running")
	case atomic.LoadUint32(&s.Status) != uint32(StateNew):
		return errors.New("this swamp is not new, use resume if it is paused")
	}
	atomic.StoreInt32(&s.runningdaemons, 0)
	s.getThisDread()
	return nil
}

/*
Pause will cease the creation of any new proxy validation operations.
   * You will be able to start the proxy pool again with Swamp.Resume()
       * when resumed it will retain the same statistics, options, and ratelimits.

   * During pause you are still able to dispense proxies.
   * Options may be changed and proxy lists may be loaded when paused.
*/
func (s *Swamp) Pause() error {
	if !s.IsRunning() {
		return errors.New("swamp is not running")
	}

	s.dbgPrint("pausing...")

	s.svcDown()
	s.svcDown()

	atomic.StoreUint32(&s.Status, uint32(StatePaused))
	return nil
}

func (s *Swamp) getThisDread() {
	go s.mapBuilder()
	<-s.conductor
	go s.jobSpawner()

	for {
		if s.IsRunning() {
			atomic.StoreUint32(&s.Status, uint32(StateRunning))
			break
		}
	}
}

// Resume will resume pause proxy pool operations, attempting to resume a running Swamp is returns an error.
func (s *Swamp) Resume() error {

	s.getThisDread()

	return nil
}
