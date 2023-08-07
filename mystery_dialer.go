package prox5

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/socks"
)

// DialContext is a simple stub adapter to implement a net.Dialer.
func (p5 *ProxyEngine) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return p5.mysteryDialer(ctx, network, addr)
}

// Dial is a simple stub adapter to implement a net.Dialer.
func (p5 *ProxyEngine) Dial(network, addr string) (net.Conn, error) {
	return p5.mysteryDialer(context.Background(), network, addr)
}

// DialTimeout is a simple stub adapter to implement a net.Dialer with a timeout.
func (p5 *ProxyEngine) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	defer cancel()
	nc, err := p5.mysteryDialer(ctx, network, addr)
	return nc, err
}

func (p5 *ProxyEngine) addTimeout(socksString string) string {
	tout := strs.Get()
	tout.MustWriteString(socksString)
	tout.MustWriteString("?timeout=")
	tout.MustWriteString(p5.GetServerTimeoutStr())
	_, _ = tout.WriteRune('s')
	socksString = tout.String()
	strs.MustPut(tout)
	return socksString
}

func (p5 *ProxyEngine) isEmpty() bool {
	stats := p5.GetStatistics()
	if stats.Checked == 0 {
		return true
	}
	if stats.Valid5+stats.Valid4+stats.Valid4a+stats.ValidHTTP == 0 {
		return true
	}
	return false
}

var ErrNoProxies = fmt.Errorf("no proxies available")

func (p5 *ProxyEngine) popSockAndLockIt(ctx context.Context) (*Proxy, error) {
	if p5.isEmpty() {
		p5.scale()
		return nil, ErrNoProxies
	}
	sock := p5.GetAnySOCKS()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context done: %w", ctx.Err())
	default:
		//
	}
	if sock == nil {
		return nil, nil
	}
	if atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		// p5.msgGotLock(socksString)
		return sock, nil
	}
	switch sock.GetProto() {
	case ProtoSOCKS5:
		p5.Valids.SOCKS5.add(sock)
	case ProtoSOCKS4:
		p5.Valids.SOCKS4.add(sock)
	case ProtoSOCKS4a:
		p5.Valids.SOCKS4a.add(sock)
	case ProtoHTTP:
		p5.Valids.HTTP.add(sock)
	default:
		return nil, fmt.Errorf("unknown protocol: %s", sock.GetProto())
	}

	return nil, nil
}

func (p5 *ProxyEngine) announceDial(network, addr string) {
	s := strs.Get()
	s.MustWriteString("prox5 dialing: ")
	s.MustWriteString(network)
	s.MustWriteString("://")
	if p5.opt.redact {
		buf.MustWriteString("[redacted]")
	} else {
		buf.MustWriteString(addr)
	}
	s.MustWriteString(addr)
	s.MustWriteString("...")
	p5.dbgPrint(s)
}

// mysteryDialer is a dialer function that will use a different proxy for every request.
// If you're looking for this function, it has been unexported. Use Dial, DialTimeout, or DialContext instead.
func (p5 *ProxyEngine) mysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	p5.announceDial(network, addr)

	if p5.isEmpty() {
		// p5.dbgPrint(simpleString("prox5: no proxies available"))
		return nil, ErrNoProxies
	}

	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	var count = 0
	for {
		max := p5.GetDialerBailout()
		switch {
		case count > max:
			return nil, fmt.Errorf("giving up after %d tries", max)
		case ctx.Err() != nil:
			return nil, fmt.Errorf("context error: %w", ctx.Err())
		case p5.conCtx.Err() != nil:
			return nil, fmt.Errorf("context closed")
		default:
			//
		}
		var sock *Proxy
		for {
			if p5.scale() {
				time.Sleep(5 * time.Millisecond)
			}
			var err error
			sock, err = p5.popSockAndLockIt(ctx)
			if err != nil {
				// println(err.Error())
				return nil, err
			}
			if sock != nil {
				break
			}
		}
		socksString := sock.String()
		var ok bool
		if sock, ok = p5.dispenseMiddleware(sock); !ok {
			atomic.StoreUint32(&sock.lock, stateUnlocked)
			p5.msgFailedMiddleware(socksString)
			continue
		}
		p5.msgTry(socksString)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(socksString)
		conn, err := dialSocks(network, addr)
		if err != nil {
			count++
			p5.msgUnableToReach(socksString, addr, err)
			continue
		}
		p5.msgUsingProxy(socksString)
		go func() {
			select {
			case <-ctx.Done():
				_ = conn.Close()
			case <-p5.conCtx.Done():
				_ = conn.Close()
			case <-p5.ctx.Done():
				_ = conn.Close()
			}
		}()
		return conn, nil
	}
}
