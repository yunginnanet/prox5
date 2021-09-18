package pxndscvm

import (
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

// GetAnySOCKS retrieves any version SOCKS proxy as a Proxy type
// Will block if one is not available!
func (s *Swamp) GetAnySOCKS() Proxy {
	for {
		select {
		case sock := <-s.Socks4:
			if !s.stillGood(sock) {
				continue
			}
			s.Stats.dispense()
			return sock
		case sock := <-s.Socks4a:
			if !s.stillGood(sock) {
				continue
			}
			s.Stats.dispense()
			return sock
		case sock := <-s.Socks5:
			if !s.stillGood(sock) {
				continue
			}
			s.Stats.dispense()
			return sock
		}
	}
}

func (s *Swamp) stillGood(candidate Proxy) bool {
	if useProx.Check(candidate) {
		s.dbgPrint(ylw+"useprox ratelimited: " + candidate.Endpoint)
		return false
	}
	if badProx.Peek(candidate) {
		s.dbgPrint(ylw+"badprox ratelimited: " + candidate.Endpoint)
		return false
	}
	if time.Since(candidate.Verified) > s.swampopt.Stale {
		s.dbgPrint("proxy stale: " + candidate.Endpoint)
		go s.Stats.stale()
		return false
	}

	return true
}

// RandomUserAgent retrieves a random user agent from our list in string form
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return randStrChoice(s.swampopt.UserAgents)
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
	return s.swampopt.Stale
}

// GetValidationTimeout returns the current value of ValidationTimeout (in seconds).
func (s *Swamp) GetValidationTimeout() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.ValidationTimeout
}

// DebugEnabled returns the current state of our Debug switch
func (s *Swamp) DebugEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.Debug
}

// GetMaxWorkers returns maximum amount of workers that validate proxies concurrently. Note this is read-only during runtime.
func (s *Swamp) GetMaxWorkers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.swampopt.MaxWorkers
}

// TODO: Implement ways to access worker pool (pond) statistics
