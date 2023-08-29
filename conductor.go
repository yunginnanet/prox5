package prox5

import (
	"context"
	"errors"
	"sync/atomic"
)

// engineState represents the current state of our ProxyEngine.
type engineState uint32

const (
	// stateRunning means the proxy pool is currently taking in proxys and validating them, and is available to dispense proxies.
	stateRunning engineState = iota
	// statePaused means the proxy pool has been with ProxyEngine.Pause() and may be resumed with ProxyEngine.Resume()
	statePaused
	// stateNew means the proxy pool has never been started.
	stateNew
)

// Start starts our proxy pool operations. Trying to start a running ProxyEngine will return an error.
func (p5 *ProxyEngine) Start() error {
	if atomic.LoadUint32(&p5.Status) != uint32(stateNew) {
		p5.DebugLogger.Printf("proxy pool has been started before, resuming instead")
		return p5.Resume()
	}
	p5.DebugLogger.Printf("starting prox5")
	p5.startDaemons()
	return nil
}

/*
Pause will cease the creation of any new proxy validation operations.
  - You will be able to start the proxy pool again with ProxyEngine.Resume(), it will have the same Statistics, options, and ratelimits.
  - During pause you are still able to dispense proxies.
  - Options may be changed and proxy lists may be loaded when paused.
  - Pausing an already paused ProxyEngine is a nonop.
*/
func (p5 *ProxyEngine) Pause() error {
	if !p5.IsRunning() {
		return errors.New("not running")
	}

	p5.dbgPrint(simpleString("pausing proxy pool"))

	// p5.quit()

	atomic.StoreUint32(&p5.Status, uint32(statePaused))

	return nil
}

func (p5 *ProxyEngine) startDaemons() {
	go p5.jobSpawner()
	atomic.StoreUint32(&p5.Status, uint32(stateRunning))
	p5.DebugLogger.Printf("prox5 started")
}

// Resume will resume pause proxy pool operations, attempting to resume a running ProxyEngine is returns an error.
func (p5 *ProxyEngine) Resume() error {
	if p5.IsRunning() {
		return errors.New("already running")
	}
	// p5.ctx, p5.quit = context.WithCancel(context.Background())
	p5.startDaemons()
	return nil
}

// CloseAllConns will close all connections in progress by the dialers (including the SOCKS server if in use).
// Note this does not effect the proxy pool, it will continue to operate as normal.
func (p5 *ProxyEngine) CloseAllConns() {
	p5.killConns()
	p5.mu.Lock()
	p5.ctx, p5.killConns = context.WithCancel(context.Background())
}
