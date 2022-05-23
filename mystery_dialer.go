package prox5

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync/atomic"

	"h12.io/socks"
)

// DialContext is a simple stub adapter for compatibility with certain packages.
func (s *Swamp) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return s.MysteryDialer(ctx, network, addr)
}

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (s *Swamp) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var (
		socksString string
		count       int
	)
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		max := s.GetDialerBailout()
		if count > max {
			return nil, errors.New("giving up after " + strconv.Itoa(max) + " tries")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var sock *Proxy
		for {
			sock = s.GetAnySOCKS(false)
			if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
				continue
			}
			break
		}
		s.dbgPrint("dialer trying: " + sock.Endpoint + "...")
		tout := ""
		if s.GetServerTimeoutStr() != "-1" {
			tout = "?timeout=" + s.GetServerTimeoutStr() + "s"
		}
		socksString = "socks" + getProtoStr(sock.proto) + "://" + sock.Endpoint + tout
		var ok bool
		if sock, ok = s.dispenseMiddleware(sock); !ok {
			s.dbgPrint(ylw + "failed middleware check, " + socksString + ", cycling..." + rst)
			continue
		}
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(socksString)
		conn, err := dialSocks(network, addr)
		if err != nil {
			count++
			s.dbgPrint(ylw + "unable to reach [redacted] with " + socksString + ", cycling..." + rst)
			continue
		}
		s.dbgPrint(grn + "MysteryDialer using socks: " + socksString + rst)
		return conn, nil
	}
}
