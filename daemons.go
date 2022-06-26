package prox5

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func (s *Swamp) svcUp() {
	atomic.AddInt32(&s.runningdaemons, 1)
}

func (s *Swamp) svcDown() {
	atomic.AddInt32(&s.runningdaemons, -1)
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
		Endpoint: sock,
		lock:     stateUnlocked,
		parent:   sm.parent,
	}

	sm.plot[sock].timesValidated.Store(0)
	sm.plot[sock].timesBad.Store(0)
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

func (s *Swamp) mapBuilder() {
	if s.pool.IsClosed() {
		s.pool.Reboot()
	}

	s.dbgPrint("map builder started")

	go func() {
		defer s.dbgPrint("map builder paused")
		for {
			select {
			case <-s.ctx.Done():
				s.svcDown()
				return
			case in := <-inChan:
				if p, ok := s.swampmap.add(in); !ok {
					continue
				} else {
					s.Pending <- p
				}
			}
		}
	}()
	s.conductor <- true
}

func (s *Swamp) recycling() int {
	if !s.GetRecyclingStatus() {
		return 0
	}

	if len(s.swampmap.plot) < 1 {
		return 0
	}

	var count int

	s.swampmap.mu.RLock()
	defer s.swampmap.mu.RUnlock()

	for _, sock := range s.swampmap.plot {
		select {
		case <-s.ctx.Done():
			return 0
		case s.Pending <- sock:
			count++
		}
	}

	return count
}

func (s *Swamp) jobSpawner() {
	if s.pool.IsClosed() {
		s.pool.Reboot()
	}

	s.dbgPrint("job spawner started")
	defer s.dbgPrint("job spawner paused")

	q := make(chan bool)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				q <- true
				s.svcDown()
				return
			case sock := <-s.Pending:
				if err := s.pool.Submit(sock.validate); err != nil {
					s.dbgPrint(ylw + err.Error() + rst)
				}
			default:
				time.Sleep(25 * time.Millisecond)
				count := s.recycling()
				s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
			}
		}
	}()

	s.svcUp()
	<-q
	s.pool.Release()
}
