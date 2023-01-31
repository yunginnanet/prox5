package prox5

import (
	"crypto/tls"
	"net/http"
	"sync"
)

func (p5 *ProxyEngine) newHTTPClient() any {
	return &http.Client{
		Transport: &http.Transport{
			DialContext:     p5.DialContext,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: p5.GetHTTPTLSVerificationStatus()},
			// TLSHandshakeTimeout:   p5.GetServerTimeout(),
			DisableKeepAlives:  true,
			DisableCompression: false,
			// MaxIdleConnsPerHost:    5,
			MaxConnsPerHost:        0,
			IdleConnTimeout:        0,
			ResponseHeaderTimeout:  0,
			ExpectContinueTimeout:  0,
			TLSNextProto:           nil,
			ProxyConnectHeader:     nil,
			GetProxyConnectHeader:  nil,
			MaxResponseHeaderBytes: 0,
			WriteBufferSize:        0,
			ReadBufferSize:         0,
			ForceAttemptHTTP2:      false,
		},
		Timeout: p5.GetServerTimeout(),
	}
}

// GetHTTPClient retrieves a pointer to an http.Client powered by mysteryDialer.
func (p5 *ProxyEngine) GetHTTPClient() *http.Client {
	if p5.httpOptsDirty.Load() {
		p5.httpClients = &sync.Pool{
			New: p5.newHTTPClient,
		}
		p5.httpOptsDirty.Store(false)
	}
	return p5.httpClients.Get().(*http.Client)
}

// RoundTrip is Mr. WorldWide. Obviously. See: https://pkg.go.dev/net/http#RoundTripper
func (p5 *ProxyEngine) RoundTrip(req *http.Request) (*http.Response, error) {
	return p5.GetHTTPClient().Do(req)
}
