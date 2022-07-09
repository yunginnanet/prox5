package prox5

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync/atomic"

	"h12.io/socks"
)

// DialContext is a simple stub adapter to implement a net.Dialer.
func (s *Swamp) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return s.MysteryDialer(ctx, network, addr)
}

// DialContext is a simple stub adapter to implement a net.Dialer.
func (s *Swamp) Dial(network, addr string) (net.Conn, error) {
	return s.DialContext(context.Background(), network, addr)
}

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (pe *ProxyEngine) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var (
		socksString string
		count       int
	)
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	for {
		max := pe.GetDialerBailout()
		if count > max {
			return nil, errors.New("giving up after " + strconv.Itoa(max) + " tries")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var sock *Proxy
		for {
			sock = pe.GetAnySOCKS(false)
			if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
				continue
			}
			break
		}
		pe.dbgPrint("dialer trying: " + sock.Endpoint + "...")
		tout := ""
		if pe.GetServerTimeoutStr() != "-1" {
			tout = "?timeout=" + pe.GetServerTimeoutStr() + "s"
		}
		socksString = "socks" + getProtoStr(sock.proto) + "://" + sock.Endpoint + tout
		var ok bool
		if sock, ok = pe.dispenseMiddleware(sock); !ok {
			pe.dbgPrint("failed middleware check, " + socksString + ", cycling...")
			continue
		}
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(socksString)
		conn, err := dialSocks(network, addr)
		if err != nil {
			count++
			pe.dbgPrint("unable to reach [redacted] with " + socksString + ", cycling...")
			continue
		}
		pe.dbgPrint("MysteryDialer using socks: " + socksString)
		return conn, nil
	}
}
