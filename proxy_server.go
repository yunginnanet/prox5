package prox5

import (
	"fmt"

	"git.tcp.direct/kayos/go-socks5"
)

type socksLogger struct {
	parent *Swamp
}

// Printf is used to handle socks server logging.
func (s socksLogger) Printf(format string, a ...interface{}) {
	s.parent.dbgPrint(fmt.Sprintf(format, a...))
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
func (s *Swamp) StartSOCKS5Server(listen, username, password string) error {
	s.socks5ServerAuth = socksCreds{username: username, password: password}

	conf := &socks5.Config{
		Credentials: s.socks5ServerAuth,
		Logger:      s.socksServerLogger,
		Dial:        s.MysteryDialer,
	}

	s.dbgPrint("listening for SOCKS5 connections on " + listen)

	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	return server.ListenAndServe("tcp", listen)
}
