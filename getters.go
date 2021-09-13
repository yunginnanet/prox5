package pxndscvm

import (
	"fmt"
	"time"
)

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
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
func (s *Swamp) GetAnySOCKS() Proxy {
	for {
		select {
		case sock := <-s.Socks4:
			if !s.stillGood(sock) {
				continue
			}
			return sock
		case sock := <-s.Socks4a:
			if !s.stillGood(sock) {
				continue
			}
			return sock
		case sock := <-s.Socks5:
			if !s.stillGood(sock) {
				continue
			}
			return sock
		}
	}
}

func (s *Swamp) stillGood(candidate Proxy) bool {
	if time.Since(candidate.Verified) > s.swampopt.Stale {
		s.dbgPrint("proxy stale: " + candidate.Endpoint)
		fmt.Println("time since: ", time.Since(candidate.Verified))
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
	return randStrChoice(s.swampopt.CheckEndpoints)
}

// DebugEnabled will return the current state of our Debug switch
func (s *Swamp) DebugEnabled() bool {
	return s.swampopt.Debug
}
