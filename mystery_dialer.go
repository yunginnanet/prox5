package prox5

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"h12.io/socks"
)

var copABuffer = &sync.Pool{New: func() interface{} { return &strings.Builder{} }}

func discardBuffer(buf *strings.Builder) {
	buf.Reset()
	copABuffer.Put(buf)
}

// DialContext is a simple stub adapter to implement a net.Dialer.
func (pe *ProxyEngine) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return pe.MysteryDialer(ctx, network, addr)
}

// Dial is a simple stub adapter to implement a net.Dialer.
func (pe *ProxyEngine) Dial(network, addr string) (net.Conn, error) {
	return pe.MysteryDialer(context.Background(), network, addr)
}

// DialTimeout is a simple stub adapter to implement a net.Dialer with a timeout.
func (pe *ProxyEngine) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
	go func() { // this is a goroutine that calls cancel() upon the deadline expiring to avoid context leaks
		<-ctx.Done()
		cancel()
	}()
	return pe.MysteryDialer(ctx, network, addr)
}

func (pe *ProxyEngine) addTimeout(socksString string) string {
	tout := copABuffer.Get().(*strings.Builder)
	tout.WriteString(socksString)
	tout.WriteString("?timeout=")
	tout.WriteString(pe.GetServerTimeoutStr())
	tout.WriteRune('s')
	socksString = tout.String()
	discardBuffer(tout)
	return socksString
}

func (pe *ProxyEngine) popSockAndLockIt(ctx context.Context) (*Proxy, error) {
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
func (pe *ProxyEngine) MysteryDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	var (
		socksString string
		count       int
	)
	// pull down proxies from channel until we get a proxy good enough for our spoiled asses

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
		if pe.GetServerTimeoutStr() != "-1" {
			socksString = pe.addTimeout(socksString)
		}
		var ok bool
		if sock, ok = pe.dispenseMiddleware(sock); !ok {
			pe.msgFailedMiddleware(socksString)
			continue
		}
		pe.msgTry(socksString)
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(socksString)
		conn, err := dialSocks(network, addr)
		if err != nil {
			count++
			pe.msgUnableToReach(socksString)
			continue
		}
		pe.msgUsingProxy(socksString)
		return conn, nil
	}
}
