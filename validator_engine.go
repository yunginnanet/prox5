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

func (pe *Swamp) prepHTTP() (*http.Client, *http.Transport, *http.Request, error) {
	req, err := http.NewRequest("GET", pe.GetRandomEndpoint(), bytes.NewBuffer([]byte("")))
	if err != nil {
		return nil, nil, nil, err
	}
	headers := make(map[string]string)
	headers["User-Agent"] = pe.RandomUserAgent()
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
		TLSHandshakeTimeout: pe.GetValidationTimeout(),
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

func (pe *Swamp) bakeHTTP(hmd *HandMeDown) (client *http.Client, req *http.Request, err error) {
	builder := pools.CopABuffer.Get().(*strings.Builder)
	builder.WriteString(hmd.protoCheck.String())
	builder.WriteString("://")
	builder.WriteString(hmd.sock.Endpoint)
	builder.WriteString("/?timeout=")
	builder.WriteString(pe.GetValidationTimeoutStr())
	builder.WriteString("s")
	dialSocks := socks.DialWithConn(builder.String(), hmd.conn)
	pools.DiscardBuffer(builder)

	var (
		purl      *url.URL
		transport *http.Transport
	)

	if client, transport, req, err = pe.prepHTTP(); err != nil {
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

func (pe *Swamp) validate(hmd *HandMeDown) (string, error) {
	var (
		client *http.Client
		req    *http.Request
		err    error
	)

	client, req, err = pe.bakeHTTP(hmd)
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

func (pe *Swamp) anothaOne() {
	pe.stats.Checked++
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

func (pe *Swamp) singleProxyCheck(sock *Proxy, protocol ProxyProtocol) error {
	defer pe.anothaOne()
	split := strings.Split(sock.Endpoint, "@")
	endpoint := split[0]
	if len(split) == 2 {
		endpoint = split[1]
	}
	conn, err := net.DialTimeout("tcp", endpoint, pe.GetValidationTimeout())
	if err != nil {
		return err
	}

	hmd := &HandMeDown{sock: sock, conn: conn, under: proxy.Direct, protoCheck: protocol}

	resp, err := pe.validate(hmd)
	if err != nil {
		pe.badProx.Check(sock)
		return err
	}

	if newip := net.ParseIP(resp); newip == nil {
		pe.badProx.Check(sock)
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

func (pe *Swamp) tally(sock *Proxy) {
	switch sock.protocol.Get() {
	case ProtoSOCKS4:
		pe.stats.v4()
		pe.Valids.SOCKS4 <- sock
	case ProtoSOCKS4a:
		pe.stats.v4a()
		pe.Valids.SOCKS4a <- sock
	case ProtoSOCKS5:
		pe.stats.v5()
		pe.Valids.SOCKS5 <- sock
	case ProtoHTTP:
		pe.stats.http()
		pe.Valids.HTTP <- sock
	default:
		return
	}
}
