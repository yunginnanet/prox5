package pxndscvm

import (
	"errors"
	"net"
	"sync/atomic"
	"time"
)

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
// Will block if one is not available!
func (s *Swamp) Socks5Str() string {
	for {
		select {
		case sock := <-s.Socks5:
			if !s.stillGood(sock) {
				continue
			}
			s.Stats.dispense()
			return sock.Endpoint
		}
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (s *Swamp) Socks4Str() string {
	for {
		select {
		case sock := <-s.Socks4:
			if !s.stillGood(sock) {
				continue
			}
			s.Stats.dispense()
			return sock.Endpoint
		}
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (s *Swamp) Socks4aStr() string {
	for {
		select {
		case sock := <-s.Socks4a:
			if !s.stillGood(sock) {
				continue
			}
			s.Stats.dispense()
			return sock.Endpoint
		}
	}
}

func (sock *Proxy) copy() (Proxy, error) {
	if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		return Proxy{Endpoint: ""}, errors.New("locked")
	}
	atomic.StoreUint32(&sock.lock, stateUnlocked)

	return Proxy{
		Endpoint:     sock.Endpoint,
		ProxiedIP:    sock.ProxiedIP,
		Proto:        sock.Proto,
		LastVerified: sock.LastVerified,
	}, nil
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
func (s *Swamp) GetAnySOCKS() Proxy {
	for {
		select {
		case sock := <-s.Socks4:
			if !s.stillGood(sock) {
				continue
			}
			if sox, err := sock.copy(); err == nil {
				s.Stats.dispense()
				return sox
			}
			continue
		case sock := <-s.Socks4a:
			if !s.stillGood(sock) {
				continue
			}
			if sox, err := sock.copy(); err == nil {
				s.Stats.dispense()
				return sox
			}
			continue
		case sock := <-s.Socks5:
			if !s.stillGood(sock) {
				continue
			}
			if sox, err := sock.copy(); err == nil {
				s.Stats.dispense()
				return sox
			}
			continue
		default:
			time.Sleep(25 * time.Millisecond)
		}
	}
}

func (s *Swamp) stillGood(sock *Proxy) bool {

	if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		return false
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if sock.TimesBad > s.GetRemoveAfter() {
		s.dbgPrint("removing proxy: " + sock.Endpoint)
		if err := s.swampmap.delete(sock.Endpoint); err != nil {
			s.dbgPrint(err.Error())
		}
	}

	if s.badProx.Peek(sock) {
		s.dbgPrint(ylw + "badProx dial ratelimited: " + sock.Endpoint + rst)
		return false
	}

	if _, err := net.DialTimeout("tcp", sock.Endpoint, time.Duration(s.GetValidationTimeout())*time.Second); err != nil {
		s.dbgPrint(ylw + sock.Endpoint + " failed dialing out during stillGood check: " + err.Error() + rst)
		return false
	}

	if time.Since(sock.LastVerified) > s.swampopt.stale {
		s.dbgPrint("proxy stale: " + sock.Endpoint)
		go s.Stats.stale()
		return false
	}

	if s.GetRecyclingStatus() {
		s.Pending <- sock
	}

	return true
}
