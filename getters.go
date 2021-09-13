package pxndscvm

import "time"

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

func (s *Swamp) getProxy() Proxy {
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
	switch {
	// if since its been validated it's ended up failing so much that its on our bad list then skip it
	case badProx.Peek(candidate):
		fallthrough
	// if we've been checking or using this too often recently then skip it
	case useProx.Check(candidate):
		fallthrough
	case time.Since(candidate.Verified) > s.swampopt.Stale:
		return false
	default:
		return true
	}
}

// RandomUserAgent retrieves a random user agent from our list in string form
func (s *Swamp) RandomUserAgent() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return RandStrChoice(s.swampopt.UserAgents)
}

// DebugEnabled will return the current state of our Debug switch
func (s *Swamp) DebugEnabled() bool {
	return s.swampopt.Debug
}
