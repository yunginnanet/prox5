package prox5

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// GetHTTPClient retrieves a pointer to an http.Client powered by MysteryDialer.
func (pe *ProxyEngine) GetHTTPClient() *http.Client {
	// var htp func(*http.Request) (*url.URL, error)
	var dctx func(ctx context.Context, network string, addr string) (net.Conn, error)
	//	if httun, htok := pe.GetHTTPTunnel(); htok {
	//		httprox, uerr := url.Parse("http://" + httun.Endpoint)
	//		if uerr == nil {
	//			htp = http.ProxyURL(httprox)
	//		}
	//	}
	// if htp == nil {
	dctx = pe.DialContext
	// }
	return &http.Client{
		Transport: &http.Transport{
			// Proxy:                 htp,
			DialContext:           dctx,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   pe.GetServerTimeout(),
			DisableKeepAlives:     true,
			DisableCompression:    false,
			MaxIdleConnsPerHost:   5,
			IdleConnTimeout:       pe.GetServerTimeout(),
			ResponseHeaderTimeout: pe.GetServerTimeout(),
		},
		Timeout: pe.GetServerTimeout(),
	}
}

// RoundTrip is Mr. WorldWide. Obviously. See: https://pkg.go.dev/net/http#RoundTripper
func (pe *ProxyEngine) RoundTrip(req *http.Request) (*http.Response, error) {
	return pe.GetHTTPClient().Do(req)
}
