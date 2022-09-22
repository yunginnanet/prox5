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
	"sync/atomic"
	"time"

	"git.tcp.direct/kayos/socks"
	"golang.org/x/net/proxy"

	"git.tcp.direct/kayos/prox5/internal/pools"
)

func (p5 *Swamp) prepHTTP() (*http.Client, *http.Transport, *http.Request, error) {
	req, err := http.NewRequest("GET", p5.GetRandomEndpoint(), bytes.NewBuffer([]byte("")))
	if err != nil {
		return nil, nil, nil, err
	}
	headers := make(map[string]string)
	headers["User-Agent"] = p5.RandomUserAgent()
	headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"
	headers["Accept-Language"] = "en-US,en;q=0.5"
	headers["'Accept-Encoding'"] = "gzip, deflate, br"
	// headers["Connection"] = "keep-alive"
	for header, value := range headers {
		req.Header.Set(header, value)
	}
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

func (p5 *Swamp) bakeHTTP(hmd *HandMeDown) (client *http.Client, req *http.Request, err error) {
	builder := pools.CopABuffer.Get().(*strings.Builder)
	builder.WriteString(hmd.protoCheck.String())
	builder.WriteString("://")
	builder.WriteString(hmd.sock.Endpoint)
	builder.WriteString("/?timeout=")
	builder.WriteString(p5.GetValidationTimeoutStr())
	builder.WriteString("s")
	dialSocks := socks.DialWithConn(builder.String(), hmd.conn)
	pools.DiscardBuffer(builder)

	var (
		purl      *url.URL
		transport *http.Transport
	)

	if client, transport, req, err = p5.prepHTTP(); err != nil {
		return
	}

	if hmd.protoCheck != ProtoHTTP {
		transport.Dial = dialSocks
		client.Transport = transport
		return
	}
	if purl, err = url.Parse("http://" + hmd.sock.Endpoint); err != nil {
		return
	}
	transport.Proxy = http.ProxyURL(purl)
	return
}

func (p5 *Swamp) validate(hmd *HandMeDown) (string, error) {
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
	if err != nil {
		return "", err
	}

	rbody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return string(rbody), err
}

func (p5 *Swamp) anothaOne() {
	p5.stats.Checked++
}

type HandMeDown struct {
	sock       *Proxy
	protoCheck ProxyProtocol
	conn       net.Conn
	under      proxy.Dialer
}

func (hmd *HandMeDown) Dial(network, addr string) (c net.Conn, err error) {
	if hmd.conn.LocalAddr().Network() != network {
		return hmd.under.Dial(network, addr)
	}
	if hmd.conn.RemoteAddr().String() != addr {
		return hmd.under.Dial(network, addr)
	}
	return hmd.conn, nil
}

func (p5 *Swamp) singleProxyCheck(sock *Proxy, protocol ProxyProtocol) error {
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

	hmd := &HandMeDown{sock: sock, conn: conn, under: proxy.Direct, protoCheck: protocol}

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

	if sock.timesValidated == 0 || sock.protocol.Get() == ProtoNull {
		// try to use the proxy with all 3 SOCKS versions
		for tryProto := range protoMap {
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
	} else {
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

func (p5 *Swamp) tally(sock *Proxy) {
	switch sock.protocol.Get() {
	case ProtoSOCKS4:
		p5.stats.v4()
		p5.Valids.SOCKS4 <- sock
	case ProtoSOCKS4a:
		p5.stats.v4a()
		p5.Valids.SOCKS4a <- sock
	case ProtoSOCKS5:
		p5.stats.v5()
		p5.Valids.SOCKS5 <- sock
	case ProtoHTTP:
		p5.stats.http()
		p5.Valids.HTTP <- sock
	default:
		return
	}
}
