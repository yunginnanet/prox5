package prox5

import (
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/common/entropy"
)

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
// Will block if one is not available!
func (s *Swamp) Socks5Str() string {
	for {
		select {
		case sock := <-s.ValidSocks5:
			if !s.stillGood(sock) {
				continue
			}
			s.stats.dispense()
			return sock.Endpoint
		}
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (s *Swamp) Socks4Str() string {
	defer s.stats.dispense()
	for {
		select {
		case sock := <-s.ValidSocks4:
			if !s.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		}
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (s *Swamp) Socks4aStr() string {
	defer s.stats.dispense()
	for {
		select {
		case sock := <-s.ValidSocks4a:
			if !s.stillGood(sock) {
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
func (s *Swamp) GetHTTPTunnel() (p *Proxy, ok bool) {
	select {
	case httptunnel := <-s.ValidHTTP:
		return httptunnel, true
	default:
		return nil, false
	}
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
// StateNew/Temporary: Pass a true boolean to this to also receive HTTP proxies.
func (s *Swamp) GetAnySOCKS(AcceptHTTP bool) *Proxy {
	defer s.stats.dispense()
	for {
		var sock *Proxy
		select {
		case sock = <-s.ValidSocks4:
			break
		case sock = <-s.ValidSocks4a:
			break
		case sock = <-s.ValidSocks5:
			break
		default:
			if !AcceptHTTP {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if httptun, htok := s.GetHTTPTunnel(); htok {
				sock = httptun
				break
			}
		}
		if s.stillGood(sock) {
			return sock
		}
		continue
	}
}

func (s *Swamp) stillGood(sock *Proxy) bool {
	for !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		entropy.RandSleepMS(200)
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if atomic.LoadInt64(&sock.timesBad) > int64(s.GetRemoveAfter()) && s.GetRemoveAfter() != -1 {
		s.dbgPrint(red + "deleting from map (too many failures): " + sock.Endpoint + rst)
		if err := s.swampmap.delete(sock.Endpoint); err != nil {
			s.dbgPrint(red + err.Error() + rst)
		}
	}

	if s.badProx.Peek(sock) {
		s.dbgPrint(ylw + "badProx dial ratelimited: " + sock.Endpoint + rst)
		return false
	}

	if time.Since(sock.lastValidated) > s.swampopt.stale {
		s.dbgPrint("proxy stale: " + sock.Endpoint)
		go s.stats.stale()
		return false
	}

	return true
}
