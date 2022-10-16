package prox5

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

type proxyMap struct {
	plot   map[string]*Proxy
	mu     *sync.RWMutex
	parent *ProxyEngine
}

func (sm proxyMap) add(sock string) (*Proxy, bool) {
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

func (sm proxyMap) exists(sock string) bool {
	if _, ok := sm.plot[sock]; !ok {
		return false
	}
	return true
}

func (sm proxyMap) delete(sock string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.exists(sock) {
		return errors.New("proxy does not exist in map")
	}

	sm.plot[sock] = nil
	delete(sm.plot, sock)
	return nil
}

func (sm proxyMap) clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for key := range sm.plot {
		delete(sm.plot, key)
	}
}

func (p5 *ProxyEngine) recycling() int {
	if !p5.GetRecyclingStatus() {
		return 0
	}

	if len(p5.proxyMap.plot) < 1 {
		return 0
	}

	var count int

	p5.proxyMap.mu.RLock()
	defer p5.proxyMap.mu.RUnlock()

	for _, sock := range p5.proxyMap.plot {
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

func (p5 *ProxyEngine) jobSpawner() {
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
				return
			case sock := <-p5.Pending:
				if err := p5.pool.Submit(sock.validate); err != nil {
					p5.dbgPrint(simpleString(err.Error()))
				}

			default:
				time.Sleep(500 * time.Millisecond)
				count := p5.recycling()
				buf := strs.Get()
				buf.MustWriteString("recycled ")
				buf.MustWriteString(strconv.Itoa(count))
				buf.MustWriteString(" proxies from our map")
				p5.dbgPrint(buf)
			}
		}
	}()

	<-q
	p5.pool.Release()
}
