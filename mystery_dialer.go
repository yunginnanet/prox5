package prox5

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/socks"

	"git.tcp.direct/kayos/prox5/internal/pools"
)

// DialContext is a simple stub adapter to implement a net.Dialer.
func (pe *Swamp) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return pe.MysteryDialer(ctx, network, addr)
}

// Dial is a simple stub adapter to implement a net.Dialer.
func (pe *Swamp) Dial(network, addr string) (net.Conn, error) {
	return pe.MysteryDialer(context.Background(), network, addr)
}

// DialTimeout is a simple stub adapter to implement a net.Dialer with a timeout.
func (pe *Swamp) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	go func() { // this is a goroutine that calls cancel() upon the deadline expiring to avoid context leaks
		<-ctx.Done()
		cancel()
	}()
	return pe.MysteryDialer(ctx, network, addr)
}

func (pe *Swamp) addTimeout(socksString string) string {
	tout := pools.CopABuffer.Get().(*strings.Builder)
	tout.WriteString(socksString)
	tout.WriteString("?timeout=")
	tout.WriteString(pe.GetServerTimeoutStr())
	tout.WriteRune('s')
	socksString = tout.String()
	pools.DiscardBuffer(tout)
	return socksString
}

func (pe *Swamp) popSockAndLockIt(ctx context.Context) (*Proxy, error) {
	sock := pe.GetAnySOCKS(false)
	socksString := sock.String()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context done: %w", ctx.Err())
	default:
		if atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
			pe.msgGotLock(socksString)
			return sock, nil
		}
		select {
		case pe.Pending <- sock:
			pe.msgCantGetLock(socksString, true)
			return nil, nil
		default:
			pe.msgCantGetLock(socksString, false)
			return nil, nil
		}
	}
}

// MysteryDialer is a dialer function that will use a different proxy for every request.
func (pe *Swamp) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses
	var count = 0
	for {
		max := pe.GetDialerBailout()
		if count > max {
			return nil, fmt.Errorf("giving up after %d tries", max)
		}
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context error: %w", err)
		}
		var sock *Proxy
		for {
			var err error
			sock, err = pe.popSockAndLockIt(ctx)
			if err != nil {
				return nil, err
			}
			if sock != nil {
				break
			}
		}
		socksString := sock.String()
		var ok bool
		if sock, ok = pe.dispenseMiddleware(sock); !ok {
			atomic.StoreUint32(&sock.lock, stateUnlocked)
			pe.msgFailedMiddleware(socksString)
			continue
		}
		pe.msgTry(socksString)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(socksString)
		conn, err := dialSocks(network, addr)
		if err != nil {
			count++
			pe.msgUnableToReach(socksString, addr, err)
			continue
		}
		pe.msgUsingProxy(socksString)
		return conn, nil
	}
}
