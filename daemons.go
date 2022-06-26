package prox5

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func (pe *ProxyEngine) svcUp() {
	atomic.AddInt32(&pe.runningdaemons, 1)
}

func (pe *ProxyEngine) svcDown() {
	atomic.AddInt32(&pe.runningdaemons, -1)
}

type swampMap struct {
	plot   map[string]*Proxy
	mu     *sync.RWMutex
	parent *ProxyEngine
}

func (sm swampMap) add(sock string) (*Proxy, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.exists(sock) {
		return nil, false
	}

	sm.plot[sock] = &Proxy{
		Endpoint: sock,
		lock:     stateUnlocked,
		parent:   sm.parent,
	}

	atomic.StoreInt64(&sm.plot[sock].timesValidated, 0)
	atomic.StoreInt64(&sm.plot[sock].timesBad, 0)
	return sm.plot[sock], true
}

func (sm swampMap) exists(sock string) bool {
	if _, ok := sm.plot[sock]; !ok {
		return false
	}
	return true
}

func (sm swampMap) delete(sock string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.exists(sock) {
		return errors.New("proxy does not exist in map")
	}

	sm.plot[sock] = nil
	delete(sm.plot, sock)
	return nil
}

func (sm swampMap) clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for key := range sm.plot {
		delete(sm.plot, key)
	}
}

func (pe *ProxyEngine) mapBuilder() {
	if pe.pool.IsClosed() {
		pe.pool.Reboot()
	}

	pe.dbgPrint("map builder started")

	go func() {
		defer pe.dbgPrint("map builder paused")
		for {
			select {
			case <-pe.ctx.Done():
				pe.svcDown()
				return
			case in := <-inChan:
				if p, ok := pe.swampmap.add(in); !ok {
					continue
				} else {
					pe.Pending <- p
				}
			}
		}
	}()
	pe.conductor <- true
}

func (pe *ProxyEngine) recycling() int {
	if !pe.GetRecyclingStatus() {
		return 0
	}

	if len(pe.swampmap.plot) < 1 {
		return 0
	}

	var count int

	pe.swampmap.mu.RLock()
	defer pe.swampmap.mu.RUnlock()

	for _, sock := range pe.swampmap.plot {
		select {
		case <-pe.ctx.Done():
			return 0
		case pe.Pending <- sock:
			count++
		}
	}

	return count
}

func (pe *ProxyEngine) jobSpawner() {
	if pe.pool.IsClosed() {
		pe.pool.Reboot()
	}

	pe.dbgPrint("job spawner started")
	defer pe.dbgPrint("job spawner paused")

	q := make(chan bool)

	go func() {
		for {
			select {
			case <-pe.ctx.Done():
				q <- true
				pe.svcDown()
				return
			case sock := <-pe.Pending:
				if err := pe.pool.Submit(sock.validate); err != nil {
					pe.dbgPrint(err.Error())
				}
			default:
				time.Sleep(25 * time.Millisecond)
				count := pe.recycling()
				pe.dbgPrint("recycled " + strconv.Itoa(count) + " proxies from our map")
			}
		}
	}()

	pe.svcUp()
	<-q
	pe.pool.Release()
}
