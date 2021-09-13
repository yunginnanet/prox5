package pxndscvm

// Socks5Str gets a SOCKS5 proxy that we have fully verified (dialed and then retrieved our IP address from a what-is-my-ip endpoint.
func (s *Swamp) Socks5Str() string {
	select {
	case sock := <-s.Socks5:
		return sock.Endpoint
	}
}

// Socks4Str gets a SOCKS4 proxy that we have fully verified.
func (s *Swamp) Socks4Str() string {
	select {
	case sock := <-s.Socks4:
		return sock.Endpoint
	}
}

// Socks4aStr gets a SOCKS4 proxy that we have fully verified.
func (s *Swamp) Socks4aStr() string {
	select {
	case sock := <-s.Socks4a:
		return sock.Endpoint
	}
}

func (s *Swamp) getProxy() *Proxy {
	select {
	case sock := <-s.Socks4:
		return sock
	case sock := <-s.Socks4a:
		return sock
	case sock := <-s.Socks5:
		return sock
	}
}
