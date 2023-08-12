package prox5

import (
	"time"

	rl "github.com/yunginnanet/Rate5"
)

// https://pkg.go.dev/github.com/yunginnanet/Rate5#Policy
var defaultUseProxyRatelimiter = rl.Policy{
	Window: 55,
	Burst:  55,
}

var defaultBadProxyRateLimiter = rl.Policy{
	Window: 55,
	Burst:  10,
}

const (
	stateUnlocked uint32 = iota
	stateLocked
)

// Proxy represents an individual proxy
type Proxy struct {
	// Endpoint is the address:port of the proxy that we connect to
	Endpoint string
	// ProxiedIP is the address that we end up having when making proxied requests through this proxy
	// TODO: parse this and store as flat int type
	ProxiedIP string
	// protocol is the version/Protocol (currently SOCKS* only) of the proxy
	protocol proto
	// lastValidated is the time this proxy was last verified working
	lastValidated time.Time
	// timesValidated is the amount of times the proxy has been validated.
	timesValidated int64
	// timesBad is the amount of times the proxy has been marked as bad.
	timesBad int64

	parent *ProxyEngine
	lock   uint32
}

// UniqueKey is an implementation of the Identity interface from Rate5.
// See: https://pkg.go.dev/github.com/yunginnanet/Rate5#Identity
func (sock *Proxy) UniqueKey() string {
	return sock.Endpoint
}

// GetProto retrieves the known protocol value of the Proxy.
func (sock *Proxy) GetProto() ProxyProtocol {
	return sock.protocol.Get()
}

// GetProto safely retrieves the protocol value of the Proxy.
func (sock *Proxy) String() string {
	buf := strs.Get()
	defer strs.MustPut(buf)
	buf.MustWriteString(sock.GetProto().String())
	buf.MustWriteString("://")
	buf.MustWriteString(sock.Endpoint)
	if sock.parent.GetServerTimeoutStr() != "-1" {
		buf.MustWriteString("?timeout=")
		buf.MustWriteString(sock.parent.GetServerTimeoutStr())
		buf.MustWriteString("s")
	}
	return buf.String()
}
