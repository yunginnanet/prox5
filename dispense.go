package prox5

import (
	"strconv"
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
		default:
			count := s.recycling()
			if count > 0 {
				s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (s *Swamp) Socks4Str() string {
	defer s.Stats.dispense()
	for {
		select {
		case sock := <-s.ValidSocks4:
			if !s.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		default:
			count := s.recycling()
			if count > 0 {
				s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
// Will block if one is not available!
func (s *Swamp) Socks4aStr() string {
	defer s.Stats.dispense()
	for {
		select {
		case sock := <-s.ValidSocks4a:
			if !s.stillGood(sock) {
				continue
			}
			return sock.Endpoint
		default:
			count := s.recycling()
			if count > 0 {
				s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
func (s *Swamp) GetAnySOCKS() *Proxy {
	defer s.Stats.dispense()
	for {
		select {
		case sock := <-s.ValidSocks4:
			if s.stillGood(sock) {
				return sock
			}
			continue
		case sock := <-s.ValidSocks4a:
			if s.stillGood(sock) {
				return sock
			}
			continue
		case sock := <-s.ValidSocks5:
			if s.stillGood(sock) {
				return sock
			}
			continue
		default:
			// s.dbgPrint(red + "no valid proxies in channels, sleeping" + rst)
			count := s.recycling()
			if count > 0 {
				s.dbgPrint(ylw + "recycled " + strconv.Itoa(count) + " proxies from our map" + rst)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Swamp) stillGood(sock *Proxy) bool {
	for !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		randSleep()
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	if sock.timesBad.Load().(int) > s.GetRemoveAfter() && s.GetRemoveAfter() != -1 {
		s.dbgPrint(red + "deleting from map (too many failures): " + sock.Endpoint + rst)
		if err := s.swampmap.delete(sock.Endpoint); err != nil {
			s.dbgPrint(red + err.Error() + rst)
		}
	}

	if s.badProx.Peek(sock) {
		s.dbgPrint(ylw + "badProx dial ratelimited: " + sock.Endpoint + rst)
		return false
	}

	if s.swampopt.stale.Load().(time.Duration) == 0 {
		return true
	}
	since := time.Since(sock.lastValidated.Load().(time.Time))
	if since > s.swampopt.stale.Load().(time.Duration) {
		s.dbgPrint("proxy stale: " + sock.Endpoint)
		go s.Stats.stale()
		return false
	}

	return true
}
