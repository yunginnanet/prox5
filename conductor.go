package prox5

import (
	"context"
	"errors"
	"sync/atomic"
)

// SwampStatus represents the current state of our ProxyEngine.
type SwampStatus uint32

const (
	// StateRunning means the proxy pool is currently taking in proxys and validating them, and is available to dispense proxies.
	StateRunning SwampStatus = iota
	// StatePaused means the proxy pool has been with ProxyEngine.Pause() and may be resumed with Swamp.Resume()
	StatePaused
	// StateNew means the proxy pool has never been started.
	StateNew
)

// Start starts our proxy pool operations. Trying to start a running ProxyEngine will return an error.
func (pe *ProxyEngine) Start() error {
	if atomic.LoadUint32(&pe.Status) != uint32(StateNew) {
		return pe.Resume()
	}
	pe.startDaemons()
	return nil
}

/*
Pause will cease the creation of any new proxy validation operations.
  - You will be able to start the proxy pool again with Swamp.Resume(), it will have the same Statistics, options, and ratelimits.
  - During pause you are still able to dispense proxies.
  - Options may be changed and proxy lists may be loaded when paused.
  - Pausing an already paused ProxyEngine is a nonop.
*/
func (pe *ProxyEngine) Pause() error {
	if !pe.IsRunning() {
		return errors.New("not running")
	}

	pe.dbgPrint(simpleString("pausing proxy pool"))

	pe.quit()

	atomic.StoreUint32(&pe.Status, uint32(StatePaused))
	return nil
}

func (pe *ProxyEngine) startDaemons() {
	go pe.mapBuilder()
	<-pe.conductor
	pe.svcUp()
	go pe.jobSpawner()

	for {
		if pe.IsRunning() {
			atomic.StoreUint32(&pe.Status, uint32(StateRunning))
			break
		}
	}
}

// Resume will resume pause proxy pool operations, attempting to resume a running ProxyEngine is returns an error.
func (pe *ProxyEngine) Resume() error {
	if pe.IsRunning() {
		return errors.New("already running")
	}
	pe.ctx, pe.quit = context.WithCancel(context.Background())
	pe.startDaemons()
	return nil
}
