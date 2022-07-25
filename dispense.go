package prox5

import (
	"strings"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
// Will block if one is not available!
func (pe *ProxyEngine) Socks5Str() string {
	for {
		select {
		case sock := <-pe.Valids.SOCKS5:
			if !pe.stillGood(sock) {
				continue
			}
			pe.stats.dispense()
			return sock.Endpoint
		}
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (pe *ProxyEngine) Socks4Str() string {
	defer pe.stats.dispense()
	for {
		select {
		case sock := <-pe.Valids.SOCKS4:
			if !pe.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		}
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (pe *ProxyEngine) Socks4aStr() string {
	defer pe.stats.dispense()
	for {
		select {
		case sock := <-pe.Valids.SOCKS4a:
			if !pe.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		}
	}
}

// GetHTTPTunnel checks for an available HTTP CONNECT proxy in our pool.
// For now, this function does not loop forever like the GetAnySOCKS does.
// Alternatively it can be included within the for loop by passing true to GetAnySOCKS.
// If there is an HTTP proxy available, ok will be true. If not, it will return false without delay.
func (pe *ProxyEngine) GetHTTPTunnel() (p *Proxy, ok bool) {
	select {
	case httptunnel := <-pe.Valids.HTTP:
		return httptunnel, true
	default:
		return nil, false
	}
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
// StateNew/Temporary: Pass a true boolean to this to also receive HTTP proxies.
func (pe *ProxyEngine) GetAnySOCKS(AcceptHTTP bool) *Proxy {
	defer pe.stats.dispense()
	for {
		var sock *Proxy
		select {
		case sock = <-pe.Valids.SOCKS4:
			break
		case sock = <-pe.Valids.SOCKS4a:
			break
		case sock = <-pe.Valids.SOCKS5:
			break
		default:
			if !AcceptHTTP {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if httptun, htok := pe.GetHTTPTunnel(); htok {
				sock = httptun
				break
			}
		}
		if pe.stillGood(sock) {
			return sock
		}
		continue
	}
}

func (pe *ProxyEngine) stillGood(sock *Proxy) bool {
	for !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		entropy.RandSleepMS(200)
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if atomic.LoadInt64(&sock.timesBad) > int64(pe.GetRemoveAfter()) && pe.GetRemoveAfter() != -1 {
		buf := copABuffer.Get().(*strings.Builder)
		buf.WriteString("deleting from map (too many failures): ")
		buf.WriteString(sock.Endpoint)
		pe.dbgPrint(buf)
		if err := pe.swampmap.delete(sock.Endpoint); err != nil {
			pe.dbgPrint(simpleString(err.Error()))
		}
	}

	if pe.badProx.Peek(sock) {
		buf := copABuffer.Get().(*strings.Builder)
		buf.WriteString("badProx dial ratelimited: ")
		buf.WriteString(sock.Endpoint)
		pe.dbgPrint(buf)
		return false
	}

	if time.Since(sock.lastValidated) > pe.swampopt.stale {
		buf := copABuffer.Get().(*strings.Builder)
		buf.WriteString("proxy stale: ")
		buf.WriteString(sock.Endpoint)
		pe.dbgPrint(buf)
		go pe.stats.stale()
		return false
	}

	return true
}
