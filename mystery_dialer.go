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
	return p5.MysteryDialer(ctx, network, addr)
}

// Dial is a simple stub adapter to implement a net.Dialer.
func (p5 *ProxyEngine) Dial(network, addr string) (net.Conn, error) {
	return p5.MysteryDialer(context.Background(), network, addr)
}

// DialTimeout is a simple stub adapter to implement a net.Dialer with a timeout.
func (p5 *ProxyEngine) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	go func() { // this is a goroutine that calls cancel() upon the deadline expiring to avoid context leaks
		<-ctx.Done()
		cancel()
	}()
	return p5.MysteryDialer(ctx, network, addr)
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

func (p5 *ProxyEngine) popSockAndLockIt(ctx context.Context) (*Proxy, error) {
	sock := p5.GetAnySOCKS()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context done: %w", ctx.Err())
	default:
		if atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
			// p5.msgGotLock(socksString)
			return sock, nil
		}
		select {
		case p5.Pending <- sock:
			// p5.msgCantGetLock(socksString, true)
			return nil, nil
		default:
			p5.msgCantGetLock(sock.String(), false)
			return nil, nil
		}
	}
}

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (p5 *ProxyEngine) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	var count = 0
	for {
		p5.scale()
		max := p5.GetDialerBailout()
		if count > max {
			return nil, fmt.Errorf("giving up after %d tries", max)
		}
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context error: %w", err)
		}
		if p5.conCtx.Err() != nil {
			return nil, fmt.Errorf("context closed")
		}
		var sock *Proxy
		for {
			var err error
			sock, err = p5.popSockAndLockIt(ctx)
			if err != nil {
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
