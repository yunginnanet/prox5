package prox5

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
)

// GetHTTPClient retrieves a pointer to an http.Client powered by MysteryDialer.
func (s *Swamp) GetHTTPClient() *http.Client {
	var htp func(*http.Request) (*url.URL, error)
	var dctx func(ctx context.Context, network string, addr string) (net.Conn, error)
	if httun, htok := s.GetHTTPTunnel(); htok {
		httprox, uerr := url.Parse("http://" + httun.Endpoint)
		if uerr == nil {
			htp = http.ProxyURL(httprox)
		}
	}
	if htp == nil {
		dctx = s.DialContext
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 htp,
			DialContext:           dctx,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   s.GetServerTimeout(),
			DisableKeepAlives:     true,
			DisableCompression:    false,
			MaxIdleConnsPerHost:   5,
			IdleConnTimeout:       s.GetServerTimeout(),
			ResponseHeaderTimeout: s.GetServerTimeout(),
		},
		Timeout: s.GetServerTimeout(),
	}
}

// RoundTrip is Mr. WorldWide. Obviously. See: https://pkg.go.dev/net/http#RoundTripper
func (s *Swamp) RoundTrip(req *http.Request) (*http.Response, error) {
	return s.GetHTTPClient().Do(req)
}
