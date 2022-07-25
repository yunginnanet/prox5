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
	go func() {
		select {
		case <-ctx.Done():
			cancel()
		}
	}()
	return pe.MysteryDialer(ctx, network, addr)
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
			return nil, fmt.Errorf("context error: %v", err)
		}
		var sock *Proxy
	popSockAndLockIt:
		for {
			sock = pe.GetAnySOCKS(false)
			socksString = sock.String()
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context done: %v", ctx.Err())
			default:
				buf := copABuffer.Get().(*strings.Builder)
				if atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
					buf.WriteString("got lock for ")
					buf.WriteString(socksString)
					break popSockAndLockIt
				}
				select {
				case pe.Pending <- sock:
					buf.WriteString("can't get lock, putting back ")
					buf.WriteString(socksString)
					pe.dbgPrint(buf)
					continue
				default:
					buf.WriteString("can't get lock, can't put back ")
					buf.WriteString(socksString)
					continue
				}
			}
		}
		buf := copABuffer.Get().(*strings.Builder)
		buf.WriteString("try dial with: ")
		buf.WriteString(sock.Endpoint)
		pe.dbgPrint(buf)
		if pe.GetServerTimeoutStr() != "-1" {
			tout := copABuffer.Get().(*strings.Builder)
			tout.WriteString("?timeout=")
			tout.WriteString(pe.GetServerTimeoutStr())
			tout.WriteRune('s')
		}
		var ok bool
		if sock, ok = pe.dispenseMiddleware(sock); !ok {
			buf := copABuffer.Get().(*strings.Builder)
			buf.WriteString("failed middleware check, ")
			buf.WriteString(sock.String())
			buf.WriteString(", cycling...")
			pe.dbgPrint(buf)
			continue
		}
		atomic.StoreUint32(&sock.lock, stateUnlocked)
		dialSocks := socks.Dial(socksString)
		conn, err := dialSocks(network, addr)
		if err != nil {
			count++
			buf := copABuffer.Get().(*strings.Builder)
			buf.WriteString("unable to reach [redacted] with ")
			buf.WriteString(socksString)
			buf.WriteString(", cycling...")
			pe.dbgPrint(buf)
			continue
		}
		buf = copABuffer.Get().(*strings.Builder)
		buf.WriteString("MysteryDialer using socks: ")
		buf.WriteString(socksString)
		pe.dbgPrint(buf)
		return conn, nil
	}
}
