package Prox5

import (
	"context"
	"crypto/tls"
	"fmt"
	"git.tcp.direct/kayos/go-socks5"
	"github.com/akutz/memconn"
	"net"
	"net/http"
	"time"
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

// StartMemoryServer starts our rotating proxy SOCKS5 server as an in-memory socket.
func (s *Swamp) StartInMemorySocks5Server() error {

	conf := &socks5.Config{
		Logger: s.socksServerLogger,
		Dial:   s.MysteryDialer,
	}

	s.dbgPrint("listening for SOCKS5 connections in memory")

	server, err := socks5.New(conf)
	if err != nil {
		return err
	}

	listener, err := memconn.Listen("memu", "Prox5")
	if err != nil {
		return err
	}
	return server.Serve(listener)
}

func (s *Swamp) GetInMemoryHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(
				ctx context.Context, _, _ string) (net.Conn, error) {
				return memconn.DialContext(ctx, "memu", "Prox5")
			},
			DisableKeepAlives:   true,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: s.swampopt.validationTimeout.Load().(time.Duration),
		},
	}
}
