package prox5

import (
	"strings"
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
	ProtoHTTPS
	ProtoSSH
)

var protoMap = map[ProxyProtocol]string{
	ProtoSOCKS5: "socks5", ProtoNull: "unknown", ProtoSOCKS4: "socks4", ProtoSOCKS4a: "socks4a",
}

var strToProto = map[string]ProxyProtocol{
	"socks5": ProtoSOCKS5, "socks4": ProtoSOCKS4, "socks4a": ProtoSOCKS4a, "http": ProtoHTTP, "ssh": ProtoSSH,
}

func protoFromStr(s string) (ProxyProtocol, bool) {
	if strings.Contains(s, "://") {
		s = strings.Split(s, "://")[0]
	}
	prot, ok := strToProto[s]
	if !ok {
		prot = ProtoNull
	}
	return prot, ok
}

var protoStrs = map[string]string{
	"socks5":  "socks5://",
	"socks4":  "socks4://",
	"socks4a": "socks4://",
	"http":    "http://",
	"https":   "https://",
	"ssh":     "ssh://",
}

func protoStrNormalize(s string) (protoStr string, cleaned string, ok bool) {
	cleaned = s
	if !strings.Contains(s, "://") {
		return
	}
	cleaned = strings.Split(cleaned, "://")[1]
	s = strings.ToLower(strings.Split(s, "://")[0])
	protoStr, ok = protoStrs[s]
	return
}

func extractProtoFromProxyString(s string) (prot ProxyProtocol, cleaned string) {
	cleaned = s
	prot, _ = protoFromStr(s)
	if prot != ProtoNull {
		cleaned = strings.Split(s, "://")[1]
	}
	return prot, cleaned
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
