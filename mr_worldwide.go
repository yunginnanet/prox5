package prox5

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// GetHTTPClient retrieves a pointer to an http.Client powered by MysteryDialer.
func (p5 *Swamp) GetHTTPClient() *http.Client {
	var dctx func(ctx context.Context, network string, addr string) (net.Conn, error)
	dctx = p5.DialContext
	return &http.Client{
		Transport: &http.Transport{
			// Proxy:                 htp,
			DialContext:           dctx,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   p5.GetServerTimeout(),
			DisableKeepAlives:     true,
			DisableCompression:    false,
			MaxIdleConnsPerHost:   5,
			IdleConnTimeout:       p5.GetServerTimeout(),
			ResponseHeaderTimeout: p5.GetServerTimeout(),
		},
		Timeout: p5.GetServerTimeout(),
	}
}

// RoundTrip is Mr. WorldWide. Obviously. See: https://pkg.go.dev/net/http#RoundTripper
func (p5 *Swamp) RoundTrip(req *http.Request) (*http.Response, error) {
	return p5.GetHTTPClient().Do(req)
}
