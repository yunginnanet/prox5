package Prox5

import (
	"errors"
	"sync/atomic"
	"time"
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
		case sock := <-s.ValidSocks4:
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
		case sock := <-s.ValidSocks4a:
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
		Endpoint:  sock.Endpoint,
		ProxiedIP: sock.ProxiedIP,
		Proto:     sock.Proto,
	}, nil
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
func (s *Swamp) GetAnySOCKS() Proxy {
	dishout := func(sock *Proxy) (Proxy, bool) {
		if !s.stillGood(sock) {
			return Proxy{}, false
		}
		if sox, err := sock.copy(); err == nil {
			s.Stats.dispense()
			return sox, true
		}
		return Proxy{}, false
	}

	for {
		var sox Proxy
		var ok bool
		select {
		case sock := <-s.ValidSocks4:
			if sox, ok = dishout(sock); ok {
				return sox
			}
			continue
		case sock := <-s.ValidSocks4a:
			if sox, ok = dishout(sock); ok {
				return sox
			}
			continue
		case sock := <-s.ValidSocks5:
			if sox, ok = dishout(sock); ok {
				return sox
			}
			continue
		default:
			s.dbgPrint(red + "no valid proxies in channels, sleeping" + rst)
			time.Sleep(10 * time.Second)
		}
	}
}

func (s *Swamp) stillGood(sock *Proxy) bool {
	for !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		time.Sleep(100 * time.Millisecond)
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if sock.timesBad.Load().(int) > s.GetRemoveAfter() {
		s.dbgPrint(red + "deleting from map (too many failures): " + sock.Endpoint + rst)
		if err := s.swampmap.delete(sock.Endpoint); err != nil {
			s.dbgPrint(err.Error())
		}
	}

	if s.badProx.Peek(sock) {
		s.dbgPrint(ylw + "badProx dial ratelimited: " + sock.Endpoint + rst)
		return false
	}

	if time.Since(sock.lastValidated.Load().(time.Time)) > s.swampopt.stale.Load().(time.Duration) {
		s.dbgPrint("proxy stale: " + sock.Endpoint)
		go s.Stats.stale()
		return false
	}

	return true
}
