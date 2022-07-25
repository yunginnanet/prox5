package prox5

import (
	"fmt"
	"git.tcp.direct/kayos/go-socks5"
	"strings"
)

type socksLogger struct {
	parent *ProxyEngine
}

// Printf is used to handle socks server logging.
func (s socksLogger) Printf(format string, a ...interface{}) {
	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString(fmt.Sprintf(format, a...))
	s.parent.dbgPrint(buf)
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
func (pe *ProxyEngine) StartSOCKS5Server(listen, username, password string) error {
	pe.socks5ServerAuth = socksCreds{username: username, password: password}

	conf := &socks5.Config{
		Credentials: pe.socks5ServerAuth,
		Logger:      pe.socksServerLogger,
		Dial:        pe.MysteryDialer,
	}

	buf := copABuffer.Get().(*strings.Builder)
	buf.WriteString("listening for SOCKS5 connections on ")
	buf.WriteString(listen)
	pe.dbgPrint(buf)

	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	return server.ListenAndServe("tcp", listen)
}
