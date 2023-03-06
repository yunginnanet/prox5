package prox5

import (
	"errors"
	"strconv"
	"time"

	"git.tcp.direct/kayos/common/entropy"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type proxyMap struct {
	plot   cmap.ConcurrentMap[string, *Proxy]
	parent *ProxyEngine
}

func (sm proxyMap) add(sock string) (*Proxy, bool) {
	sm.plot.SetIfAbsent(sock, &Proxy{
		Endpoint:       sock,
		protocol:       newImmutableProto(),
		lastValidated:  time.UnixMilli(0),
		timesValidated: 0,
		timesBad:       0,
		parent:         sm.parent,
		lock:           stateUnlocked,
	})

	return sm.plot.Get(sock)
}

func (sm proxyMap) delete(sock string) error {
	if _, ok := sm.plot.Get(sock); !ok {
		return errors.New("proxy not found")
	}
	sm.plot.Remove(sock)
	return nil
}

func (sm proxyMap) clear() {
	sm.plot.Clear()
}

func (p5 *ProxyEngine) recycling() int {
	if !p5.recycleMu.TryLock() {
		return 0
	}
	defer p5.recycleMu.Unlock()

	switch {
	case !p5.GetRecyclingStatus(), p5.proxyMap.plot.Count() < 1:
		return 0
	default:
		select {
		case <-p5.recycleTimer.C:
			break
		default:
			return 0
		}
	}

	var count = 0

	switch p5.GetRecyclerShuffleStatus() {
	case true:
		var tuples []cmap.Tuple[string, *Proxy]
		for tuple := range p5.proxyMap.plot.IterBuffered() {
			tuples = append(tuples, tuple)
		}
		entropy.GetOptimizedRand().Shuffle(len(tuples), func(i, j int) {
			tuples[i], tuples[j] = tuples[j], tuples[i]
		})
		for _, tuple := range tuples {
			p5.Pending.add(tuple.Val)
			count++
		}
	case false:
		for tuple := range p5.proxyMap.plot.IterBuffered() {
			p5.Pending.add(tuple.Val)
			count++
		}
	}
	redu
	return count
}

func (p5 *ProxyEngine) jobSpawner() {
	p5.pool.Reboot()

	p5.dbgPrint(simpleString("job spawner started"))

	q := make(chan bool, 1)

	go func() {
		for {
			if !p5.IsRunning() {
				q <- true
				return
			}
			// select {
			// case <-p5.ctx.Done():
			// default:
			// }
			if p5.Pending.Len() < 1 {
				count := p5.recycling()
				switch {
				case count > 0:
					buf := strs.Get()
					buf.MustWriteString("recycled ")
					buf.MustWriteString(strconv.Itoa(count))
					buf.MustWriteString(" proxies from our map")
					p5.dbgPrint(buf)
				default:
					time.Sleep(time.Millisecond * 100)
				}
				continue
			}

			var sock *Proxy

			p5.Pending.Lock()
			switch p5.GetRecyclingStatus() {
			case true:
				el := p5.Pending.Front()
				p5.Pending.MoveToBack(el)
				sock = el.Value.(*Proxy)
			}
			p5.Pending.Unlock()

			_ = p5.scale()
			if err := p5.pool.Submit(sock.validate); err != nil {
				p5.dbgPrint(simpleString(err.Error()))
			}

		}
	}()

	<-q
	p5.dbgPrint(simpleString("job spawner paused"))
	p5.pool.Release()
}
