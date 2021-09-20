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
		return Proxy{}, errors.New("locked")
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)
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
			var sox Proxy
			var err error
			if sox, err = sock.copy(); err != nil {
				continue
			}
			s.Stats.dispense()
			return sox
		case sock := <-s.Socks4a:
			if !s.stillGood(sock) {
				continue
			}
			var sox Proxy
			var err error
			if sox, err = sock.copy(); err != nil {
				continue
			}
			s.Stats.dispense()
			return sox
		case sock := <-s.Socks5:
			if !s.stillGood(sock) {
				continue
			}
			var sox Proxy
			var err error
			if sox, err = sock.copy(); err != nil {
				continue
			}
			s.Stats.dispense()
			return sox
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

	return true
}

// RandomUserAgent retrieves a random user agent from our list in string form
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.userAgents)
}

// GetRandomEndpoint returns a random whatismyip style endpoint from our Swamp's options
func (s *Swamp) GetRandomEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.CheckEndpoints)
}

// GetStaleTime returns the duration of time after which a proxy will be considered "stale".
func (s *Swamp) GetStaleTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.stale
}

// GetValidationTimeout returns the current value of validationTimeout (in seconds).
func (s *Swamp) GetValidationTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.validationTimeout
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (s *Swamp) GetMaxWorkers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.maxWorkers
}

// IsRunning returns true if our background goroutines defined in daemons.go are currently operational
func (s *Swamp) IsRunning() bool {
	if s.runningdaemons == 2 {
		return true
	}
	return false
}

// TODO: Implement ways to access worker pool (pond) statistics
