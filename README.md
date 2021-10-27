## Prox5

[![GoDoc](https://godoc.org/git.tcp.direct/kayos/Prox5?status.svg)](https://godoc.org/git.tcp.direct/kayos/Prox5) [![Go Report Card](https://goreportcard.com/badge/github.com/yunginnanet/Prox5)](https://goreportcard.com/report/github.com/yunginnanet/Prox5) [![IRC](https://img.shields.io/badge/ircd.chat-%23tcpdirect-blue.svg)](ircs://ircd.chat:6697/#tcpdirect)

### SOCKS5/4/4a validating proxy pool

![Demo](./Prox5.gif)

This package is for managing, validating, and accessing thousands upon thousands of arbitrary SOCKS proxies.

Notably it features a SOCKS5 server function that dials out from a different validated proxy for every connection.

This project is in development. It works and has been used in "production", but mainly this readme and the documentation needs some love.
Please break it and let me know what broke.

**See [the docs](https://godoc.org/git.tcp.direct/kayos/Prox5) and the [example](example/main.go) for more details.**
