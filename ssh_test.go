package prox5

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/yunginnanet/go-spew/spew"
	"golang.org/x/crypto/ssh"
)

const (
	testUser = "yeetersonmcgee"
	testPass = "yeetinemallday"
)

type testSSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
	t        *testing.T
	errChan  chan error
	closed   *atomic.Bool
	testHTTP *atomic.Value
}

func newTestSSHServer(t *testing.T) *testSSHServer {
	s := &testSSHServer{
		t:        t,
		errChan:  make(chan error, 1), // Buffered channel to handle non-blocking error reporting
		closed:   &atomic.Bool{},
		testHTTP: &atomic.Value{},
	}
	key := genKey()
	signer := signerFromKey(key)

	s.config = &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == testUser && string(password) == testPass {
				return nil, nil
			}
			return nil, ssh.ErrNoAuth
		},
	}

	s.config.AddHostKey(signer)

	s.closed.Store(false)

	return s
}

func genKey() *rsa.PrivateKey {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	return k
}

func signerFromKey(key *rsa.PrivateKey) ssh.Signer {
	signer, _ := ssh.NewSignerFromKey(key)
	return signer
}

func (s *testSSHServer) start() string {

	var err error
	if s.listener, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
		s.t.Fatal(err)
	}

	go s.handler()

	return s.listener.Addr().String()
}

func (s *testSSHServer) handler() {
	for {
		conn, err := s.listener.Accept()
		if err != nil && !strings.Contains(err.Error(), "use of closed") {
			s.errChan <- err // Send the error to the channel
			return
		}
		go s.handleConnection(conn)
	}
}

type tcpIPRequest struct {
	HostToConnect       string
	PortToConnect       uint32
	OriginatorIPAddress string
	OriginatorPort      uint32
}

func (s *testSSHServer) testHTTPServer() string {
	if s.testHTTP.Load() != nil {
		return s.testHTTP.Load().(string)
	}
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<b>yeet</b>"))
	}))
	s.testHTTP.Store(serv.URL)
	s.t.Logf("test HTTP server started on %s", serv.URL)
	s.t.Cleanup(serv.Close)
	return serv.URL
}

func (s *testSSHServer) handleDirectTCPIP(req *ssh.Request, channel io.ReadWriteCloser, reply bool) {
	// direct-tcpip request data structure as per RFC 4254, section 7.2

	data := &tcpIPRequest{}

	if req == nil {
		return
	}

	if err := ssh.Unmarshal(req.Payload, data); err != nil {
		s.t.Errorf("Failed to unmarshal direct-tcpip request: %v", err)
		channel.Close()
		return
	}

	s.t.Logf("direct-tcpip request: %+v", data)

	srvURL := s.testHTTPServer()

	s.t.Logf("faking connection to remote host: %s:%d", data.HostToConnect, data.PortToConnect)

	prt, _ := strconv.Atoi(strings.Split(strings.TrimPrefix(srvURL, "http://"), ":")[1])
	data.HostToConnect = strings.TrimSuffix(strings.Split(srvURL, "://")[1], fmt.Sprintf(":%d", prt))
	data.PortToConnect = uint32(prt)

	remoteConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", data.HostToConnect, data.PortToConnect))
	if err != nil {
		s.t.Logf("Failed to connect to remote host: %s:%d, error: %v", data.HostToConnect, data.PortToConnect, err)
		if reply {
			_ = req.Reply(false, nil)
		}
		_ = channel.Close()
		return
	}

	s.t.Logf("connected to remote host: %s", remoteConn.RemoteAddr())

	if reply {
		s.t.Logf("replying to request")
		if err = req.Reply(true, nil); err != nil {
			s.t.Errorf("Failed to reply to request: %v", err)
			_ = channel.Close()
			_ = remoteConn.Close()
			return
		}
	}

	go func() {
		defer func() {
			_ = channel.Close()
			_ = remoteConn.Close()
		}()
		if _, err = io.Copy(channel, remoteConn); err != nil && !strings.Contains(err.Error(), "use of closed") {
			s.t.Errorf("failed to copy from remote to channel: %v", err)
		}
	}()
	go func() {
		defer func() {
			_ = channel.Close()
			_ = remoteConn.Close()
		}()
		if _, err = io.Copy(remoteConn, channel); err != nil && !strings.Contains(err.Error(), "use of closed") {
			s.t.Errorf("failed to copy from channel to remote: %v", err)
		}
	}()
}

func (s *testSSHServer) handleConnection(conn net.Conn) {
	if conn == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		if conn != nil {
			_ = conn.Close()
		}
	}()
	clientConn, channels, requests, err := ssh.NewServerConn(conn, s.config)
	if err != nil && !strings.Contains(err.Error(), "use of closed") {
		s.t.Logf("failed to establish server connection: %v", err)
		return
	}

	if clientConn == nil {
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-requests:
				s.t.Log(spew.Sdump(req))
			case newChannel := <-channels:
				s.t.Logf("new channel: %s", newChannel.ChannelType())
				switch newChannel.ChannelType() {
				case "direct-tcpip":
					tcpIPChan, tcpIPReqs, chanErr := newChannel.Accept()
					if chanErr != nil {
						s.t.Errorf("failed to accept direct-tcpip channel: %v", chanErr)
						return
					}
					s.t.Logf("accepted direct-tcpip channel")
					if len(newChannel.ExtraData()) > 0 {
						go s.handleDirectTCPIP(&ssh.Request{
							Type:      "direct-tcpip",
							WantReply: true,
							Payload:   newChannel.ExtraData(),
						}, tcpIPChan, false)
					}
					go func() {
						for {
							select {
							case <-ctx.Done():
								return
							case req := <-tcpIPReqs:
								go s.handleDirectTCPIP(req, tcpIPChan, false)
							}
						}
					}()

				default:
					s.t.Errorf("unhandled channel type: %s", newChannel.ChannelType())
					if err = newChannel.Reject(ssh.UnknownChannelType, "unhandled channel type"); err != nil {
						s.t.Errorf("failed to reject channel: %v", err)
					}
					_ = clientConn.Close()
				}
			}
		}
	}()

	s.t.Logf("new connection from %s", clientConn.RemoteAddr())

	if err := clientConn.Wait(); err != nil {
		s.t.Logf("failed to wait for client connection: %v", err)
	}
}

func (s *testSSHServer) stop() {
	if err := s.listener.Close(); err != nil {
		s.t.Errorf("failed to close listener: %v", err)
	}
	select {
	case err := <-s.errChan:
		s.t.Logf("server stopped with error: %v", err)
	default:
	}
	close(s.errChan)
}

func TestSSHDialer(t *testing.T) {
	t.Run("TestSuccessfulConnection", func(t *testing.T) {
		server := newTestSSHServer(t)
		serverAddr := server.start()
		defer server.stop()

		dialer := NewPasswordSSHDialer(serverAddr, testUser, testPass)
		conn, err := dialer.Dial("tcp", "google.com:80")
		if err != nil {
			t.Fatalf("failed to establish connection: %v", err)
		}

		var n int
		if n, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: google.com\r\n\r\n")); err != nil {
			t.Fatalf("failed to write to connection: %v", err)
		}

		t.Logf("[client] wrote %d bytes to connection", n)

		buf := make([]byte, 1024)
		n, err = conn.Read(buf)
		if err != nil {
			t.Fatalf("failed to read from connection: %v", err)
		}
		t.Logf("[client] read %d bytes from connection", n)
		t.Log(string(buf[:n]))
		if !strings.Contains(string(buf[:n]), "<b>yeet</b>") {
			t.Fatal("expected response to contain '<b>yeet</b>'")
		}
		if err = conn.Close(); err != nil {
			t.Fatalf("failed to close connection: %v", err)
		}
	})
	t.Run("TestFailedAuthentication", func(t *testing.T) {
		server := newTestSSHServer(t)
		serverAddr := server.start()
		defer server.stop()

		dialer := NewPasswordSSHDialer(serverAddr, testUser, "yeet5")
		conn, err := dialer.Dial("tcp", "google.com:80")
		if err == nil {
			if conn != nil {
				if err = conn.Close(); err != nil {
					t.Errorf("failed to close connection: %v", err)
				}
			}
			t.Fatalf("expected authentication error, got none")
		}
	})
}
