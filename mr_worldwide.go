package prox5

import (
	"crypto/tls"
	"net/http"
)

// GetHTTPClient retrieves a pointer to an http.Client powered by MysteryDialer.
func (p5 *ProxyEngine) GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext:     p5.DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// TLSHandshakeTimeout:   p5.GetServerTimeout(),
			DisableKeepAlives:   true,
			DisableCompression:  false,
			MaxIdleConnsPerHost: 5,
			// IdleConnTimeout:       p5.GetServerTimeout(),
			// ResponseHeaderTimeout: p5.GetServerTimeout(),
		},
		Timeout: p5.GetServerTimeout(),
	}
}

// RoundTrip is Mr. WorldWide. Obviously. See: https://pkg.go.dev/net/http#RoundTripper
func (p5 *ProxyEngine) RoundTrip(req *http.Request) (*http.Response, error) {
	return p5.GetHTTPClient().Do(req)
}
