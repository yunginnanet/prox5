package Prox5

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func (s *Swamp) svcUp() {
	running := s.runningdaemons.Load().(int)
	s.runningdaemons.Store(running + 1)
}

func (s *Swamp) svcDown() {
	running := s.runningdaemons.Load().(int)
	s.quit <- true
	s.runningdaemons.Store(running - 1)
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
	for !atomic.CompareAndSwapUint32(&sm.plot[sock].lock, stateUnlocked, stateLocked) {
		randSleep()
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

func (s *Swamp) mapBuilder() {
	var filtered string
	var ok bool

	s.dbgPrint("map builder started")
	defer s.dbgPrint("map builder paused")

	go func() {
		for {
			select {
			case in := <-inChan:
				if filtered, ok = s.filter(in); !ok {
					continue
				}
				if p, ok := s.swampmap.add(filtered); !ok {
					continue
				} else {
					s.Pending <- p
				}
			default:
				//
			}
		}
	}()
	s.conductor <- true
	s.svcUp()
	<-s.quit
}

func (s *Swamp) recycling() int {
	if !s.GetRecyclingStatus() {
		return 0
	}

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
	if s.pool.IsClosed() {
		s.pool.Reboot()
	}
	s.dbgPrint("job spawner started")
	defer s.dbgPrint("job spawner paused")

	q := make(chan bool)

	go func() {
		for {
			select {
			case <-s.quit:
				q <- true
				return
			case sock := <-s.Pending:
				if err := s.pool.Submit(sock.validate); err != nil {
					s.dbgPrint(ylw + err.Error() + rst)
				}
			default:
				time.Sleep(1 * time.Second)
				count := s.recycling()
				s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
			}
		}
	}()

	s.svcUp()
	<-q
	s.pool.Release()
}
