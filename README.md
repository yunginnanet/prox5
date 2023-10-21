
<div align="center"><h1>Prox5</h1>

### SOCKS5/4/4a validating proxy pool + SOCKS5 server

<img alt="Animated Screenshot of Prox5 Example" height=250 width=500 src="https://tcp.ac/i/3WRfz.gif" />

[![GoDoc](https://godoc.org/git.tcp.direct/kayos/prox5?status.svg)](https://pkg.go.dev/git.tcp.direct/kayos/prox5) [![Go Report Card](https://goreportcard.com/badge/github.com/yunginnanet/prox5)](https://goreportcard.com/report/github.com/yunginnanet/prox5) [![IRC](https://img.shields.io/badge/ircd.chat-%23tcpdirect-blue.svg)](ircs://ircd.chat:6697/#tcpdirect) [![Test Status](https://github.com/yunginnanet/prox5/actions/workflows/go.yml/badge.svg)](https://github.com/yunginnanet/prox5/actions/workflows/go.yml) ![five](https://img.shields.io/badge/fhjones-55555-blue)

`import git.tcp.direct/kayos/prox5`

Prox5 is a golang library for managing, validating, and utilizing a very large amount of arbitrary SOCKS proxies.

Notably it features interface compatible dialer functions that dial out from different proxies for every connection, and a SOCKS5 server that utilizes those functions.

---
> **Warning**
> **Using prox5 to proxy connections from certain offsec tools tends to cause denial of servics.**
>
> ### e.g: https://youtu.be/qVRFnxjD7o8
>
>
> Please spazz out responsibly.
---

</div>

## Table of Contents

   1. [Overview](#overview)
      1. [Validation Engine](#validation-engine)
      1. [Auto Scaler](#auto-scaler)
      1. [Rate Limiting](#rate-limiting)
      1. [Accessing Validated Proxies](#accessing-validated-proxies)
   1. [Additional info](#additional-info)
      1. [The Secret Sauce](#the-secret-sauce)
      1. [External Integrations](#external-integrations)
   1. [Status and Final Thoughts](#status-and-final-thoughts)

---

## Overview

### Validation Engine

  1) TCP Dial to the endpoint, if successful, reuse net.Conn down for step 2
  2) HTTPS GET request to a list of IP echo endpoints
  3) Store the IP address discovered during step 2
  4) Allocate a new `prox5.Proxy` type || update an existing one, store latest info
  5) Enqueue a pointer to this `proxy.Proxy` instance, instantiating it for further use

### Auto Scaler

The validation has an optional auto scale feature that allows for the automatic tuning of validation worker count as more proxies are dispensed. 
This feature is still new, but seems to work well. It can be enabled with `[...].EnableAutoScaler()`. 

Please refer to the autoscale related items within [the documentation](https://pkg.go.dev/git.tcp.direct/kayos/prox5) for more info.

### Rate Limiting

Using [Rate5](https://github.com/yunginnanet/Rate5), prox5 naturally reduces the frequency of proxies that fail to validate. It does this by reducing the frequency proxies are accepted into the validation pipeline the more they fail to verify or fail to successfully connect to an endpoint. This is not yet adjustable, but will be soon. See [the documentation for Rate5](https://pkg.go.dev/git.tcp.direct/kayos/rate5), and the source code for this project (defs.go is a good place to start) for more info.

### Accessing Validated Proxies

 - Retrieve validated 4/4a/5 proxies as simple strings for generic use
 - Use one of the dialer functions with any golang code that calls for a net.Dialer
 - Spin up a SOCKS5 server that will then make rotating use of your validated proxies

---

## Additional info

### The Secret Sauce

What makes Prox5 special is largely the Mystery Dialer. This dialer satisfies the net.Dialer and ContextDialer interfaces. The implementation is a little bit different from your average dialer. Here's roughly what happens when you dial out with a ProxyEngine;

- Loads up a previously verified proxy
- Attempts to make connection with the dial endpoint using said proxy
- Upon failure, prox5:
  - repeats this process *mid-dial*
  - does not drop connection to the client
- Once a proxy has been successfully used to connect to the target endpoint, prox5 passes the same net.Conn onto the client

### External Integrations

<details>
  <summary>Mullvad</summary>


Take a look at [mullsox](https://git.tcp.direct/kayos/mullsox) for an easy way to access all of the mullvad proxies reachable from any one VPN endpoint. It is trivial to feed the results of `GetAndVerifySOCKS` into prox5. 

Here's a snippet that should just about get you there:

```golang
package main

import (
    "os"
    "time"

    "git.tcp.direct/kayos/mullsox"
    "git.tcp.direct/kayos/prox5"
)

func main() {
	p5 := prox5.NewProxyEngine()
	mc := mullsox.NewChecker()

	if err := mc.Update(); err != nil {
		println(err.Error())
		return
	}

	incoming, _ := mc.GetAndVerifySOCKS()

	var count = 0
	for line := range incoming {
		if p5.LoadSingleProxy(line.String()) {
			count++
		}
	}

	if count == 0 {
		println("failed to load any proxies")
		return
	}

	if err := p5.Start(); err != nil {
		println(err.Error())
		return
	}
	
	go func() {
		if err := p5.StartSOCKS5Server("127.0.0.1:42069", "", ""); err != nil {
			println(err.Error())
			os.Exit(1)
		}
	}()
	
	time.Sleep(time.Millisecond * 500)
	
	println("proxies loaded and socks server started")
}
```
</details>

<details>
  <summary>ProxyBonanza</summary>

Take a look at [ProxyGonanza](https://git.tcp.direct/kayos/proxygonanza)

_(TODO: code example here)_

</details>

---

## Status and Final Thoughts

**This project is in development.** 

It "works" and has been used in "production", but still needs some love.

Please break it and let me know what broke.


The way you choose to use this lib is yours. The API is fairly extensive for you to be able to customize runtime configuration without having to do any surgery.

Things like the amount of validation workers that are concurrently operating, timeouts, and proxy re-use policies may be tuned in real-time. 

---

<div align="center">

# **Please see [the docs](https://pkg.go.dev/git.tcp.direct/kayos/prox5) and the [example](example/main.go) for more details.**
	
</div>
