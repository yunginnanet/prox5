package Prox5

import (
	"errors"
	"strconv"
	"strings"
	"sync"
)


func (s *Swamp) svcUp() {
	s.mu.Lock()
	s.runningdaemons++
	s.conductor <- true
	s.mu.Unlock()
}

func (s *Swamp) svcDown() {
	s.mu.Lock()
	s.runningdaemons--
	s.mu.Unlock()
}

func (s *Swamp) svcStatus() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runningdaemons
}

type swampMap struct {
	plot   map[string]*Proxy
	mu     *sync.RWMutex
	parent *Swamp
}

func (sm swampMap) add(sock string) (*Proxy, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var auth = &proxyAuth{}

	if strings.Contains(sock, "@") {
		split := strings.Split(sock, "@")
		sock = split[1]
		authsplit := strings.Split(split[0], ":")
		auth.username = authsplit[0]
		auth.password = authsplit[1]
	}

	if sm.exists(sock) {
		return nil, false
	}
	p := &Proxy{
		Endpoint: sock,
		lock:     stateUnlocked,
		parent:   sm.parent,
	}
	p.timesValidated.Store(0)
	p.timesBad.Store(0)
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
	defer func() {
		s.svcDown()
		s.dbgPrint("map builder paused")
	}()

	s.dbgPrint("map builder started")

	for {
		select {
		case in := <-inChan:
			if filtered, ok = filter(in); !ok {
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
			//
		}
	}
}

func (s *Swamp) recycling() int {
	if !s.GetRecyclingStatus() {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.swampmap.mu.RLock()
	if len(s.swampmap.plot) < 1 {
		s.swampmap.mu.RUnlock()
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
	s.swampmap.mu.RUnlock()
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
		case sock := <-s.Pending:
			if err := s.pool.Submit(sock.validate); err != nil {
				s.dbgPrint(ylw + err.Error() + rst)
			}
		case <-s.quit:
			return
		default:
			count := s.recycling()
			s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
		}
	}
}
