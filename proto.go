package prox5

import (
	"sync"
	"sync/atomic"

	"git.tcp.direct/kayos/common/pool"
)

type ProxyProtocol int8

const (
	// ProtoNull is a null value for ProxyProtocol.
	ProtoNull ProxyProtocol = iota
	ProtoSOCKS4
	ProtoSOCKS4a
	ProtoSOCKS5
	ProtoHTTP
)

var protoMap = map[ProxyProtocol]string{
	ProtoSOCKS5: "socks5", ProtoNull: "unknown", ProtoSOCKS4: "socks4", ProtoSOCKS4a: "socks4a",
}

func (p ProxyProtocol) String() string {
	return protoMap[p]
}

type proto struct {
	proto *atomic.Value
	// immutable
	*sync.Once
}

func newImmutableProto() proto {
	p := proto{
		proto: &atomic.Value{},
		Once:  &sync.Once{},
	}
	p.proto.Store(ProtoNull)
	return p
}

func (p *proto) Get() ProxyProtocol {
	return p.proto.Load().(ProxyProtocol)
}

func (p *proto) set(proxyproto ProxyProtocol) {
	p.Do(func() {
		p.proto.Store(proxyproto)
	})
}

func (p ProxyProtocol) writeProtoString(builder *pool.String) {
	builder.MustWriteString(p.String())
}

func (p ProxyProtocol) writeProtoURI(builder *pool.String) {
	p.writeProtoString(builder)
	builder.MustWriteString("://")
}
