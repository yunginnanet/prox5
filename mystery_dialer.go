package Prox5

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"h12.io/socks"
)

func (s *Swamp) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return s.MysteryDialer(ctx, network, addr)
}

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (s *Swamp) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var sock Proxy
	var socksString string
	var conn net.Conn
	var count int
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		max := s.GetDialerBailout()
		if count > max {
			return nil, errors.New("giving up after " + strconv.Itoa(max) + " tries")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sock = s.GetAnySOCKS()
		s.dbgPrint("dialer trying: " + sock.Endpoint + "...")
		var err error
		socksString = fmt.Sprintf("socks%s://%s?timeout=%ss", sock.GetProto(), sock.Endpoint, s.GetTimeoutSecondsStr())
		dialSocks := socks.Dial(socksString)
		if conn, err = dialSocks(network, addr); err != nil {
			count++
			s.dbgPrint(ylw + "unable to reach [redacted] with " + socksString + ", cycling..." + rst)
			continue
		}
		break
	}
	s.dbgPrint(grn + "MysteryDialer using socks: " + socksString + rst)
	return conn, nil
}
