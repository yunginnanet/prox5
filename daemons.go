package pxndscvm

import "time"

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
			s.mu.Lock()
			s.swampmap[in] = &Proxy{
				Endpoint: in,
			}
			s.mu.Unlock()
		case <- s.quit:
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
