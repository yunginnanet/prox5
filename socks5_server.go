package prox5

import (
	"strings"

	"git.tcp.direct/kayos/go-socks5"

	"git.tcp.direct/kayos/prox5/internal/pools"
)

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
func (pe *Swamp) StartSOCKS5Server(listen, username, password string) error {

	conf := &socks5.Config{
		Credentials: socksCreds{username: username, password: password},
		Logger:      pe.DebugLogger,
		Dial:        pe.MysteryDialer,
		// Resolver:    pe.MysteryResolver,
	}

	buf := pools.CopABuffer.Get().(*strings.Builder)
	buf.WriteString("listening for SOCKS5 connections on ")
	buf.WriteString(listen)
	pe.dbgPrint(buf)

	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	return server.ListenAndServe("tcp", listen)
}
