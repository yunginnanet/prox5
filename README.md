# Prox5

[![GoDoc](https://godoc.org/git.tcp.direct/kayos/prox5?status.svg)](https://pkg.go.dev/git.tcp.direct/kayos/prox5) [![Go Report Card](https://goreportcard.com/badge/github.com/yunginnanet/prox5)](https://goreportcard.com/report/github.com/yunginnanet/prox5) [![IRC](https://img.shields.io/badge/ircd.chat-%23tcpdirect-blue.svg)](ircs://ircd.chat:6697/#tcpdirect)

### SOCKS5/4/4a validating proxy pool + server

  
Prox5 is a golang library for managing, validating, and accessing thousands upon thousands of arbitrary SOCKS proxies.

Notably it features interface compatible dialer functions that dial out from different proxies for every connection, and a SOCKS5 server that utilizes those functions.

---

### Initial Validation Sequence

- TCP Dial to the endpoint
- HTTPS GET request to a list of IP echo endpoints

Prox5 will then store the endpoint's outward appearing IP address and mark it as valid for use.

### The Secret Sauce

What makes Prox5 special is largely the Mystery Dialer. This dialer satisfies the net.Dialer interface. Upon using the dialer to connect to and endpoint, Prox5:

- Loads up a previously verified proxy
- Attempts to make connection with the dial endpoint using said proxy
- Upon failure, prox5:
  - repeats this process *mid-dial*
  - does not drop connection to the client
- Once a proxy has been successfully used to connect to the target endpoint, prox5 passes the same net.Conn onto the client

### Accessing Validated Proxies

 
 - Retrieve validated 4/4a/5 proxies as simple strings for generic use
 - Use one of the dialer functions with any golang code that calls for a net.Dialer
 - Spin up a SOCKS5 server that will then make rotating use of your validated proxies
 

 
The way you choose to use this lib is yours. The API is fairly extensive for you to be able to customize runtime configuration without having to do any surgery.
  
Things like the amount of validation workers that are concurrently operating, timeouts, and proxy re-use policies may be tuned in real-time. [please read the docs.](https://pkg.go.dev/git.tcp.direct/kayos/Prox5)

 ---
 
**This project is in development.** 

It "works" and has been used in "production", but still needs some love.

Please break it and let me know what broke.

### **See [the docs](https://pkg.go.dev/git.tcp.direct/kayos/Prox5) and the [example](example/main.go) for more details.**
