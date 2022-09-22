package randtls

import (
	"context"
	"crypto/tls"
	"net"

	uhttp "github.com/ooni/oohttp"
	utls "github.com/refraction-networking/utls"
)

// See: https://github.com/ooni/oohttp/blob/main/example/example-utls/tls.go

type adapter struct {
	*utls.UConn
	conn net.Conn
}

// Asserts that we follow the interface.
var _ uhttp.TLSConn = &adapter{}

// ConnectionState implements the tls.ConnectionState interface.
func (c *adapter) ConnectionState() tls.ConnectionState {
	ustate := c.UConn.ConnectionState()
	return tls.ConnectionState{
		Version:                     ustate.Version,
		HandshakeComplete:           ustate.HandshakeComplete,
		DidResume:                   ustate.DidResume,
		CipherSuite:                 ustate.CipherSuite,
		NegotiatedProtocol:          ustate.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  ustate.NegotiatedProtocolIsMutual,
		ServerName:                  ustate.ServerName,
		PeerCertificates:            ustate.PeerCertificates,
		VerifiedChains:              ustate.VerifiedChains,
		SignedCertificateTimestamps: ustate.SignedCertificateTimestamps,
		OCSPResponse:                ustate.OCSPResponse,
		TLSUnique:                   ustate.TLSUnique,
	}
}

// HandshakeContext implements TLSConn's HandshakeContext.
func (c *adapter) HandshakeContext(ctx context.Context) error {
	errch := make(chan error, 1)
	go func() {
		errch <- c.UConn.Handshake()
	}()
	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NetConn implements TLSConn's NetConn
func (c *adapter) NetConn() net.Conn {
	return c.conn
}

// utlsFactory creates a new uTLS connection.
func utlsFactory(conn net.Conn, config *tls.Config) uhttp.TLSConn {
	uConfig := &utls.Config{
		RootCAs:                     config.RootCAs,
		NextProtos:                  config.NextProtos,
		ServerName:                  config.ServerName,
		InsecureSkipVerify:          config.InsecureSkipVerify,
		DynamicRecordSizingDisabled: config.DynamicRecordSizingDisabled,
	}
	return &adapter{
		UConn: utls.UClient(conn, uConfig, utls.HelloFirefox_55),
		conn:  conn,
	}
}
