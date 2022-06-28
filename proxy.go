package prox5

import (
	"time"

	rl "github.com/yunginnanet/Rate5"
)

// https://pkg.go.dev/github.com/yunginnanet/Rate5#Policy
var defUseProx = rl.Policy{
	Window: 60,
	Burst:  2,
}

var defBadProx = rl.Policy{
	Window: 60,
	Burst:  3,
}

const (
	stateUnlocked uint32 = iota
	stateLocked
)

type ProxyProtocol uint8

const (
	protoNULL ProxyProtocol = iota
	ProtoSOCKS4
	ProtoSOCKS4a
	ProtoSOCKS5
	ProtoHTTP
)

// Proxy represents an individual proxy
type Proxy struct {
	// Endpoint is the address:port of the proxy that we connect to
	Endpoint string
	// ProxiedIP is the address that we end up having when making proxied requests through this proxy
	ProxiedIP string
	// proto is the version/Protocol (currently SOCKS* only) of the proxy
	proto ProxyProtocol
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
