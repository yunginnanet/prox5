package prox5

import (
	"crypto/tls"
	"net/http"
)

// GetHTTPClient retrieves a pointer to an http.Client powered by MysteryDialer.
func (s *Swamp) GetHTTPClient() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	s.httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext:         s.DialContext,
			DisableKeepAlives:   true,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: s.GetServerTimeout(),
		},
		Timeout: s.GetServerTimeout(),
	}
	return s.httpClient
}

// RoundTrip is Mr. WorldWide. Obviously. See: https://pkg.go.dev/net/http#RoundTripper
func (s *Swamp) RoundTrip(req *http.Request) (*http.Response, error) {
	return s.GetHTTPClient().Do(req)
}
