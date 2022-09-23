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

func (p5 *Swamp) svcUp() {
	atomic.AddInt32(&p5.runningdaemons, 1)
}

func (p5 *Swamp) svcDown() {
	atomic.AddInt32(&p5.runningdaemons, -1)
}

type swampMap struct {
	plot   map[string]*Proxy
	mu     *sync.RWMutex
	parent *Swamp
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

func (p5 *Swamp) mapBuilder() {
	if p5.pool.IsClosed() {
		p5.pool.Reboot()
	}

	p5.dbgPrint(simpleString("map builder started"))

	go func() {
		defer p5.dbgPrint(simpleString("map builder paused"))
		for {
			select {
			case <-p5.ctx.Done():
				p5.svcDown()
				return
			case in := <-inChan:
				if p, ok := p5.swampmap.add(in); !ok {
					continue
				} else {
					p5.Pending <- p
				}
			default:
				time.Sleep(500 * time.Millisecond)
				p5.recycling()
			}
		}
	}()
	p5.conductor <- true
}

func (p5 *Swamp) recycling() int {
	if !p5.GetRecyclingStatus() {
		return 0
	}

	if len(p5.swampmap.plot) < 1 {
		return 0
	}

	var count int

	p5.swampmap.mu.RLock()
	defer p5.swampmap.mu.RUnlock()

	for _, sock := range p5.swampmap.plot {
		select {
		case <-p5.ctx.Done():
			return 0
		case p5.Pending <- sock:
			count++
		default:
			continue
		}
	}

	return count
}

func (p5 *Swamp) jobSpawner() {
	if p5.pool.IsClosed() {
		p5.pool.Reboot()
	}

	p5.dbgPrint(simpleString("job spawner started"))
	defer p5.dbgPrint(simpleString("job spawner paused"))

	q := make(chan bool)

	go func() {
		for {
			select {
			case <-p5.ctx.Done():
				q <- true
				p5.svcDown()
				return
			case sock := <-p5.Pending:
				if err := p5.pool.Submit(sock.validate); err != nil {
					p5.dbgPrint(simpleString(err.Error()))
				}

			default:
				time.Sleep(500 * time.Millisecond)
				count := p5.recycling()
				buf := pools.CopABuffer.Get().(*strings.Builder)
				buf.WriteString("recycled ")
				buf.WriteString(strconv.Itoa(count))
				buf.WriteString(" proxies from our map")
				p5.dbgPrint(buf)
			}
		}
	}()

	p5.svcUp()
	<-q
	p5.pool.Release()
}
