## PxndScvm
[![GoDoc](https://godoc.org/github.com/yunginnanet/pxndscvm?status.svg)](https://godoc.org/github.com/yunginnanet/pxndscvm) [![Go Report Card](https://goreportcard.com/badge/github.com/yunginnanet/HellPot)](https://goreportcard.com/report/github.com/yunginnanet/pxndscvm)
### SOCKS5/4/4a validating proxy pool

This package will load a list of SOCKS proxies and continuously validate them and provide functions for retrieving validated proxies. Pipe it a file filled with SOCKS proxies (host:port per line) and it will validate them continously and automatically weed out the invalid ones.

This project is in development.

**See [the docs](https://godoc.org/git.tcp.direct/kayos/pxndscvm) and the [example](example/main.go) for more details.**
