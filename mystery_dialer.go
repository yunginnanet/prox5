package prox5

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"h12.io/socks"
)

// DialContext is a simple stub adapter to implement a net.Dialer with context.
func (s *Swamp) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return s.MysteryDialer(ctx, network, addr)
}

// Dial is a simple stub adapter to implement a net.Dialer.
func (s *Swamp) Dial(network, addr string) (net.Conn, error) {
	return s.DialContext(context.Background(), network, addr)
}

// DialTimeout is a simple stub adapter to implement a net.Dialer with a timeout.
func (s *Swamp) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	return s.MysteryDialer(ctx, network, addr)
}

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (s *Swamp) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var sock *Proxy
	var conn net.Conn
	var count int
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		max := s.GetDialerBailout()
		if count > max {
			return nil, errors.New("giving up after " + strconv.Itoa(max) + " tries")
		}
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context error: %v", err)
		}

		sock = s.GetAnySOCKS()
		for !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
			if sock == nil {
				break
			}
			randSleep()
		}
		if sock == nil {
			continue
		}

		s.dbgPrint("dialer trying: " + sock.Endpoint + "...")
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(sock.String())
		var err error
		if conn, err = dialSocks(network, addr); err != nil {
			count++
			s.dbgPrint(ylw + "unable to reach [redacted] with " + sock.String() + ", cycling..." + rst)
			continue
		}
		break
	}
	s.dbgPrint(grn + "MysteryDialer using socks: " + sock.String() + rst)
	return conn, nil
}
