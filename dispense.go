package prox5

import (
	"sync/atomic"
	"time"
)

func (p5 *ProxyEngine) getSocksStr(proto ProxyProtocol) string {
	var sock *Proxy
	var list *proxyList
	switch proto {
	case ProtoSOCKS4:
		list = &p5.Valids.SOCKS4
	case ProtoSOCKS4a:
		list = &p5.Valids.SOCKS4a
	case ProtoSOCKS5:
		list = &p5.Valids.SOCKS5
	case ProtoHTTP:
		list = &p5.Valids.HTTP
	}
	for {
		if list.Len() == 0 {
			p5.recycling()
			time.Sleep(250 * time.Millisecond)
			continue
		}
		list.Lock()
		sock = list.Remove(list.Front()).(*Proxy)
		list.Unlock()
		switch {
		case sock == nil:
			p5.recycling()
			time.Sleep(250 * time.Millisecond)
			continue
		case !p5.stillGood(sock):
			sock = nil
			continue
		default:
			p5.stats.dispense()
			return sock.Endpoint
		}
	}
}

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
// Will block if one is not available!
func (p5 *ProxyEngine) Socks5Str() string {
	return p5.getSocksStr(ProtoSOCKS5)
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (p5 *ProxyEngine) Socks4Str() string {
	return p5.getSocksStr(ProtoSOCKS4)
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (p5 *ProxyEngine) Socks4aStr() string {
	return p5.getSocksStr(ProtoSOCKS4a)
}

// GetHTTPTunnel checks for an available HTTP CONNECT proxy in our pool.
func (p5 *ProxyEngine) GetHTTPTunnel() string {
	return p5.getSocksStr(ProtoHTTP)
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
func (p5 *ProxyEngine) GetAnySOCKS() *Proxy {
	var sock *Proxy
	defer p5.stats.dispense()

	for {
		select {
		case <-p5.ctx.Done():
			return nil
		default:
			//
		}
		for _, list := range p5.Valids.Slice() {
			list.RLock()
			if list.Len() > 0 {
				list.RUnlock()
				sock = list.pop()
				switch {
				case sock == nil:
					p5.recycling()
					time.Sleep(50 * time.Millisecond)
				case p5.stillGood(sock):
					return sock
				default:
					sock = nil
				}
				continue
			}
			list.RUnlock()
		}
	}
}

func (p5 *ProxyEngine) stillGood(sock *Proxy) bool {
	if sock == nil {
		return false
	}
	if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		return false
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if p5.GetRemoveAfter() != -1 && atomic.LoadInt64(&sock.timesBad) > int64(p5.GetRemoveAfter()) {
		buf := strs.Get()
		buf.MustWriteString("deleting from map (too many failures): ")
		buf.MustWriteString(sock.Endpoint)
		p5.dbgPrint(buf)
		if err := p5.proxyMap.delete(sock.Endpoint); err != nil {
			p5.dbgPrint(simpleString(err.Error()))
		}
	}

	if p5.badProx.Peek(sock) {
		buf := strs.Get()
		buf.MustWriteString("badProx dial ratelimited: ")
		buf.MustWriteString(sock.Endpoint)
		p5.dbgPrint(buf)
		return false
	}

	if time.Since(sock.lastValidated) > p5.opt.stale {
		buf := strs.Get()
		buf.MustWriteString("proxy stale: ")
		buf.MustWriteString(sock.Endpoint)
		p5.dbgPrint(buf)
		p5.stats.stale()
		return false
	}

	return true
}
