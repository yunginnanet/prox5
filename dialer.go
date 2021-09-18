package pxndscvm

import (
	"context"
	"net"
	"time"

	"h12.io/socks"
)

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (s *Swamp) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var sock Proxy
	sock = Proxy{Endpoint: ""}
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		time.Sleep(10 * time.Millisecond)
		candidate := s.GetAnySOCKS()
		if !s.stillGood(candidate) {
			continue
		}

		sock = candidate

		if sock.Endpoint != "" {
			break
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var dialSocks = socks.Dial("socks" + sock.Proto + "://" + sock.Endpoint + "?timeout=10s")

	return dialSocks(network, addr)
}
