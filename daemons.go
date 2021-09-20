package pxndscvm

import (
	"sync"
	"time"
)

func (s *Swamp) svcUp() {
	s.mu.Lock()
	s.runningdaemons++
	s.mu.Unlock()
}

func (s *Swamp) svcDown() {
	s.mu.Lock()
	s.runningdaemons--
	s.mu.Unlock()
}

type swampMap struct {
	plot map[string]*Proxy
	mu   *sync.Mutex
}

func (sm swampMap) add(sock string) (*Proxy, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.exists(sock) {
		return nil, false
	}
	p := &Proxy{Endpoint: sock, lock: stateUnlocked}
	sm.plot[sock]=p
	return p, true
}

func (sm swampMap) exists(sock string) bool {
	if _, ok := sm.plot[sock]; !ok {
		return false
	}
	return true
}

func (sm swampMap) clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.plot = make(map[string]*Proxy)
}

func (s *Swamp) mapBuilder() {
	s.svcUp()
	s.dbgPrint("map builder started")
	defer func() {
		s.svcDown()
		s.dbgPrint("map builder paused")
	}()
	for {
		select {
		case in := <-inChan:
			if !s.stage1(in) {
				continue
			}
			if p, ok := s.swampmap.add(in); !ok {
				continue
			} else {
				s.Pending <- p
			}
		case <-s.quit:
			return
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (s *Swamp) jobSpawner() {
	s.svcUp()
	s.dbgPrint("job spawner started")
	defer func() {
		s.svcDown()
		s.dbgPrint("job spawner paused")
	}()
	for {
		if s.Status == Paused {
			return
		}
		select {
		case <-s.quit:
			return
		default:
			go s.pool.Submit(s.validate)
			time.Sleep(time.Duration(10) * time.Millisecond)
		}
	}
}
