package prox5

import (
	"git.tcp.direct/kayos/go-socks5"

	"git.tcp.direct/kayos/common/pool"
)

var strs = pool.NewStringFactory()

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

	conf := &socks5.Config{
		Credentials: socksCreds{username: username, password: password},
		Logger:      p5.DebugLogger,
		Dial:        p5.MysteryDialer,
		// Resolver:    pe.MysteryResolver,
	}

	buf := strs.Get()
	buf.MustWriteString("listening for SOCKS5 connections on ")
	buf.MustWriteString(listen)
	p5.dbgPrint(buf)

	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	return server.ListenAndServe("tcp", listen)
}
