package pxndscvm

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

func (s *Swamp) svcUp() {
	s.runningdaemons++
}

func (s *Swamp) svcDown() {
	s.runningdaemons--
}

type swampMap struct {
	plot   map[string]*Proxy
	mu     *sync.Mutex
	parent *Swamp
}

func (sm swampMap) add(sock string) (*Proxy, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.exists(sock) {
		return nil, false
	}
	p := &Proxy{
		Endpoint:       sock,
		lock:           stateUnlocked,
		TimesValidated: 0,
		TimesBad:       0,
		parent:         sm.parent,
	}
	sm.plot[sock] = p
	return p, true
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
	delete(sm.plot, sock)
	return nil
}

func (sm swampMap) clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.plot = make(map[string]*Proxy)
}

func (s *Swamp) mapBuilder() {
	var filtered string
	var ok bool
	s.svcUp()
	s.dbgPrint("map builder started")
	defer func() {
		s.svcDown()
		s.dbgPrint("map builder paused")
	}()
	for {
		select {
		case in := <-inChan:
			if filtered, ok = s.stage1(in); !ok {
				continue
			}
			if p, ok := s.swampmap.add(filtered); !ok {
				continue
			} else {
				s.Pending <- p
			}
		case <-s.quit:
			return
		default:
			count := s.recycling()
			s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
		}
	}
}

func (s *Swamp) recycling() int {
	if !s.GetRecyclingStatus() {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.swampmap.plot) < 1 {
		return 0
	}
	var count int
	for _, sock := range s.swampmap.plot {
		select {
		case s.Pending <- sock:
			count++
		default:
			continue
		}
	}
	return count
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
		case sock := <-s.Pending:
			if err := s.pool.Submit(sock.validate); err != nil {
				s.dbgPrint(ylw+err.Error()+rst)
			}
			time.Sleep(time.Duration(10) * time.Millisecond)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
