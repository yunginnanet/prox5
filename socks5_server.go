package prox5

import (
	"sync"

	"git.tcp.direct/kayos/go-socks5"

	"git.tcp.direct/kayos/common/pool"
)

var strs = pool.NewStringFactory()

type cpool struct {
	*sync.Pool
}

var bufs = cpool{
	Pool: &sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024)
		},
	},
}

func (c cpool) Get() []byte {
	return c.Pool.Get().([]byte)
}

func (c cpool) Put(cc []byte) {
	c.Pool.Put(cc)
}

type socksCreds struct {
	username string
	password string
}

// Valid implements the socks5.CredentialStore interface.
func (s socksCreds) Valid(username, password string) bool {
	if s.username == username && s.password == password {
		return true
	}
	return false
}

// StartSOCKS5Server starts our rotating proxy SOCKS5 server.
// listen is standard Go listen string, e.g: "127.0.0.1:1080".
// username and password are used for authenticatig to the SOCKS5 server.
func (p5 *ProxyEngine) StartSOCKS5Server(listen, username, password string) error {
	cator := socks5.UserPassAuthenticator{Credentials: socks5.StaticCredentials{username: password}}
	server := socks5.NewServer(
		socks5.WithBufferPool(bufs),
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
		socks5.WithLogger(p5.DebugLogger),
		socks5.WithDial(p5.MysteryDialer),
		// socks5.WithGPool(p5.pool),
	)

	buf := strs.Get()
	buf.MustWriteString("listening for SOCKS5 connections on ")
	buf.MustWriteString(listen)
	p5.dbgPrint(buf)

	return server.ListenAndServe("tcp", listen)
}
