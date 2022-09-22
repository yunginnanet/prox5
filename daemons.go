package prox5

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/prox5/internal/pools"
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
		Endpoint:       sock,
		protocol:       newImmutableProto(),
		lastValidated:  time.UnixMilli(0),
		timesValidated: 0,
		timesBad:       0,
		parent:         sm.parent,
		lock:           stateUnlocked,
	}

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

	pe.dbgPrint(simpleString("map builder started"))

	go func() {
		defer pe.dbgPrint(simpleString("map builder paused"))
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
			default:
				pe.recycling()
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
		default:
			continue
		}
	}

	return count
}

func (pe *ProxyEngine) jobSpawner() {
	if pe.pool.IsClosed() {
		pe.pool.Reboot()
	}

	pe.dbgPrint(simpleString("job spawner started"))
	defer pe.dbgPrint(simpleString("job spawner paused"))

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
					pe.dbgPrint(simpleString(err.Error()))
				}
			default:
				time.Sleep(25 * time.Millisecond)
				count := pe.recycling()
				buf := pools.CopABuffer.Get().(*strings.Builder)
				buf.WriteString("recycled ")
				buf.WriteString(strconv.Itoa(count))
				buf.WriteString(" proxies from our map")
				pe.dbgPrint(buf)
			}
		}
	}()

	pe.svcUp()
	<-q
	pe.pool.Release()
}
