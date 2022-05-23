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
	s.quit <- true
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

	return sm.plot[sock], true
}

func (sm swampMap) exists(sock string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
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

	sm.plot = make(map[string]*Proxy)
}

func (s *Swamp) build(stop chan struct{}) bool {
	select {
	case in := <-inChan:
		p, ok := s.swampmap.add(in)
		if !ok {
			break
		}
		s.Pending <- p
	case <-stop:
		return false
	default:
		time.Sleep(250 * time.Millisecond)
	}
	return true
}

func (s *Swamp) mapBuilder() {
	stop := make(chan struct{})
	s.dbgPrint("map builder started")
	defer s.dbgPrint("map builder paused")

	go func() {
		for {
			if !s.build(stop) {
				return
			}
		}
	}()
	s.conductor <- true
	s.svcUp()
	<-s.quit
	stop <- struct{}{}
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
		case s.Pending <- sock:
			count++
			time.Sleep(250 * time.Millisecond)
		default:
			time.Sleep(1 * time.Second)
			continue
		}
	}

	return count
}

func (s *Swamp) employ(stop chan struct{}) bool {
	select {
	case <-s.quit:
		stop <- struct{}{}
		return false
	case sock := <-s.Pending:
		if err := s.pool.Submit(sock.validate); err != nil {
			s.dbgPrint(ylw + err.Error() + rst)
		}
	default:
		time.Sleep(25 * time.Millisecond)
		count := s.recycling()
		s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
	}
	return true
}
func (s *Swamp) jobSpawner() {
	if s.pool.IsClosed() {
		s.pool.Reboot()
	}

	s.dbgPrint("job spawner started")
	defer s.dbgPrint("job spawner paused")

	stop := make(chan struct{})

	go func() {
		for {
			if !s.employ(stop) {
				return
			}
		}
	}()

	s.svcUp()
	<-stop
	s.pool.Release()
}
