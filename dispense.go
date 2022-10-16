package prox5

import (
	"sync/atomic"
	"time"
)

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
// Will block if one is not available!
func (p5 *Swamp) Socks5Str() string {
	for {
		select {
		case sock := <-p5.Valids.SOCKS5:
			if !p5.stillGood(sock) {
				continue
			}
			p5.stats.dispense()
			return sock.Endpoint
		default:
			p5.recycling()
		}
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (p5 *Swamp) Socks4Str() string {
	defer p5.stats.dispense()
	for {
		select {
		case sock := <-p5.Valids.SOCKS4:
			if !p5.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		default:
			p5.recycling()
		}
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (p5 *Swamp) Socks4aStr() string {
	defer p5.stats.dispense()
	for {
		select {
		case sock := <-p5.Valids.SOCKS4a:
			if !p5.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		default:
			p5.recycling()
		}
	}
}

// GetHTTPTunnel checks for an available HTTP CONNECT proxy in our pool.
// For now, this function does not loop forever like the GetAnySOCKS does.
// Alternatively it can be included within the for loop by passing true to GetAnySOCKS.
// If there is an HTTP proxy available, ok will be true. If not, it will return false without delay.
func (p5 *Swamp) GetHTTPTunnel() (p *Proxy, ok bool) {
	select {
	case httptunnel := <-p5.Valids.HTTP:
		return httptunnel, true
	default:
		return nil, false
	}
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
func (p5 *Swamp) GetAnySOCKS() *Proxy {
	defer p5.stats.dispense()
	for {
		var sock *Proxy
		select {
		case sock = <-p5.Valids.SOCKS4:
			break
		case sock = <-p5.Valids.SOCKS4a:
			break
		case sock = <-p5.Valids.SOCKS5:
			break
		default:
			p5.recycling()
		}
		if p5.stillGood(sock) {
			return sock
		}
		continue
	}
}

func (p5 *Swamp) stillGood(sock *Proxy) bool {
	if sock == nil {
		return false
	}
	if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		return false
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if atomic.LoadInt64(&sock.timesBad) > int64(p5.GetRemoveAfter()) && p5.GetRemoveAfter() != -1 {
		buf := strs.Get()
		buf.MustWriteString("deleting from map (too many failures): ")
		buf.MustWriteString(sock.Endpoint)
		p5.dbgPrint(buf)
		if err := p5.swampmap.delete(sock.Endpoint); err != nil {
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

	if time.Since(sock.lastValidated) > p5.swampopt.stale {
		buf := strs.Get()
		buf.MustWriteString("proxy stale: ")
		buf.MustWriteString(sock.Endpoint)
		p5.dbgPrint(buf)
		go p5.stats.stale()
		return false
	}

	return true
}
