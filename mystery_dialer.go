package pxndscvm

import (
	"context"
	"fmt"
	"net"

	"h12.io/socks"
)

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (s *Swamp) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var sock Proxy
	var socksString string
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		sock = s.GetAnySOCKS(false)
		s.dbgPrint("dialer trying: " + sock.Endpoint + "...")

		socksString = fmt.Sprintf("socks%s://%s?timeout=%ds", sock.Proto, sock.Endpoint, s.GetValidationTimeout())
		dialSocks := socks.Dial(socksString)
		if _, err := dialSocks("tcp", addr); err != nil {
			s.dbgPrint(ylw + "unable to reach [redacted] with SOCKS: " + sock.Endpoint + ", cycling..." + rst)
			continue
		}
		break
	}

	dialSocks := socks.Dial(socksString)
	s.dbgPrint(grn+"MysteryDialer using socks: " + socksString+rst)
	return dialSocks(network, addr)
}
