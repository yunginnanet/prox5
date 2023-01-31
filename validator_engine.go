package prox5

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/socks"
	"golang.org/x/net/proxy"
)

var headerPool = sync.Pool{
	New: func() interface{} {
		hdr := make(http.Header)
		hdr["User-Agent"] = []string{""}
		hdr["Accept"] = []string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"}
		hdr["Accept-Language"] = []string{"en-US,en;q=0.5"}
		hdr["Accept-Encoding"] = []string{"gzip, deflate, br"}
		return hdr
	},
}

func (p5 *ProxyEngine) prepHTTP() (*http.Client, *http.Transport, *http.Request, error) {
	req, err := http.NewRequest("GET", p5.GetRandomEndpoint(), bytes.NewBuffer([]byte("")))
	if err != nil {
		return nil, nil, nil, err
	}
	headers := headerPool.Get().(http.Header)
	headers["User-Agent"] = []string{p5.RandomUserAgent()}

	var client = &http.Client{}
	var transporter = &http.Transport{
		DisableKeepAlives:   true,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		TLSHandshakeTimeout: p5.GetValidationTimeout(),
	}

	return client, transporter, req, err
}

func (sock *Proxy) bad() {
	atomic.AddInt64(&sock.timesBad, 1)
}

func (sock *Proxy) good() {
	atomic.AddInt64(&sock.timesValidated, 1)
	sock.lastValidated = time.Now()
}

func httpEndpoint(hmd *handMeDown) (func(r *http.Request) (*url.URL, error), error) {
	s := strs.Get()
	defer strs.MustPut(s)
	s.MustWriteString("http://")
	s.MustWriteString(hmd.sock.Endpoint)
	purl, err := url.Parse(s.String())
	if err != nil {
		return nil, err
	}
	return http.ProxyURL(purl), nil
}

func (p5 *ProxyEngine) bakeHTTP(hmd *handMeDown) (client *http.Client, req *http.Request, err error) {
	builder := strs.Get()
	builder.MustWriteString(hmd.protoCheck.String())
	builder.MustWriteString("://")
	builder.MustWriteString(hmd.sock.Endpoint)
	builder.MustWriteString("/?timeout=")
	builder.MustWriteString(p5.GetValidationTimeoutStr())
	builder.MustWriteString("s")
	dialSocks := socks.DialWithConn(builder.String(), hmd.conn)
	strs.MustPut(builder)

	var transport *http.Transport

	client, transport, req, err = p5.prepHTTP()
	if err != nil {
		if req != nil && req.Header != nil {
			headerPool.Put(req.Header)
		}
		return
	}

	if hmd.protoCheck != ProtoHTTP {
		transport.Dial = dialSocks
		client.Transport = transport
		return
	}

	proxyURL, err := httpEndpoint(hmd)
	if err != nil {
		if req != nil && req.Header != nil {
			headerPool.Put(req.Header)
		}
		return
	}

	transport.Proxy = proxyURL
	return
}

func (p5 *ProxyEngine) validate(hmd *handMeDown) (string, error) {
	var (
		client *http.Client
		req    *http.Request
		err    error
	)

	client, req, err = p5.bakeHTTP(hmd)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	defer func() {
		if req != nil && req.Header != nil {
			headerPool.Put(req.Header)
		}
	}()
	if err != nil {
		return "", err
	}

	rbody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return string(rbody), err
}

func (p5 *ProxyEngine) anothaOne() {
	atomic.AddInt64(&p5.stats.Checked, 1)
}

type handMeDown struct {
	sock       *Proxy
	protoCheck ProxyProtocol
	conn       net.Conn
	under      proxy.Dialer
}

func (hmd *handMeDown) Dial(network, addr string) (c net.Conn, err error) {
	if hmd.conn.LocalAddr().Network() != network {
		return hmd.under.Dial(network, addr)
	}
	if hmd.conn.RemoteAddr().String() != addr {
		return hmd.under.Dial(network, addr)
	}
	return hmd.conn, nil
}

func (p5 *ProxyEngine) singleProxyCheck(sock *Proxy, protocol ProxyProtocol) error {
	defer p5.anothaOne()
	split := strings.Split(sock.Endpoint, "@")
	endpoint := split[0]
	if len(split) == 2 {
		endpoint = split[1]
	}
	conn, err := net.DialTimeout("tcp", endpoint, p5.GetValidationTimeout())
	if err != nil {
		return err
	}

	hmd := &handMeDown{sock: sock, conn: conn, under: proxy.Direct, protoCheck: protocol}

	resp, err := p5.validate(hmd)
	if err != nil {
		p5.badProx.Check(sock)
		return err
	}

	if newip := net.ParseIP(resp); newip == nil {
		p5.badProx.Check(sock)
		return errors.New("bad response from http request: " + resp)
	}

	sock.ProxiedIP = resp

	return nil
}

func (sock *Proxy) validate() {
	if !atomic.CompareAndSwapUint32(&sock.lock, stateUnlocked, stateLocked) {
		return
	}
	defer atomic.StoreUint32(&sock.lock, stateUnlocked)

	select {
	case <-sock.parent.ctx.Done():
		return
	default:
	}

	pe := sock.parent
	if pe.useProx.Check(sock) {
		// s.dbgPrint("useProx ratelimited: " + sock.Endpoint )
		return
	}

	// determined as bad, won't try again until it expires from that cache
	if pe.badProx.Peek(sock) {
		pe.msgBadProxRate(sock)
		return
	}

	// TODO: consider giving the option for verbose logging of this stuff?

	switch {
	case sock.timesValidated == 0, sock.protocol.Get() == ProtoNull:
		// try to use the proxy with all 3 SOCKS versions
		for tryProto := range protoMap {
			if tryProto == ProtoNull {
				continue
			}
			select {
			case <-pe.ctx.Done():
				return
			default:
				if err := pe.singleProxyCheck(sock, tryProto); err != nil {
					// if the proxy is no good, we continue on to the next.
					continue
				}
				sock.protocol.set(tryProto)
				break
			}
		}
	default:
		if err := pe.singleProxyCheck(sock, sock.GetProto()); err != nil {
			sock.bad()
			pe.badProx.Check(sock)
			return
		}
	}

	switch sock.protocol.Get() {
	case ProtoSOCKS4, ProtoSOCKS4a, ProtoSOCKS5, ProtoHTTP:
		pe.msgChecked(sock, true)
	default:
		pe.msgChecked(sock, false)
		sock.bad()
		pe.badProx.Check(sock)
		return
	}

	sock.good()
	pe.tally(sock)
}

func (p5 *ProxyEngine) tally(sock *Proxy) bool {
	var target proxyList
	switch sock.protocol.Get() {
	case ProtoSOCKS4:
		p5.stats.v4()
		target = p5.Valids.SOCKS4
	case ProtoSOCKS4a:
		p5.stats.v4a()
		target = p5.Valids.SOCKS4a
	case ProtoSOCKS5:
		p5.stats.v5()
		target = p5.Valids.SOCKS5
	case ProtoHTTP:
		p5.stats.http()
		target = p5.Valids.HTTP
	default:
		return false
	}
	target.Lock()
	target.PushBack(sock)
	target.Unlock()
	return true
}
