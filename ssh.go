package prox5

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type SSHDialer struct {
	host         string
	clientConfig *ssh.ClientConfig
	clientConn   *ssh.Client
	timeout      time.Duration
	mu           sync.RWMutex
}

func ioClose(closer io.Closer) {
	_ = closer.Close() // we don't care about this error. consider it "handled"
}

func getSignersFromSocket(uri string) (signers []ssh.Signer, err error) {
	if strings.Contains(uri, "://") {
		if uriSplit := strings.Split(uri, "://"); len(uriSplit) == 2 {
			uri = uriSplit[1]
		}
	}
	var conn net.Conn
	if conn, err = net.Dial("unix", uri); err != nil {
		return nil, fmt.Errorf("failed to connect to ssh-agent: %w", err)
	}
	defer ioClose(conn)
	sshAgent := agent.NewClient(conn)
	if signers, err = sshAgent.Signers(); err != nil {
		return nil, fmt.Errorf("failed to get signers from ssh-agent socket: %w", err)
	}
	if len(signers) == 0 {
		return nil, errors.New("no signers provided by ssh-agent socket")
	}
	return signers, nil
}

func NewPasswordSSHDialer(endpoint string, user, pass string) *SSHDialer {
	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	clientConfig.SetDefaults()
	return &SSHDialer{
		clientConfig: clientConfig,
		host:         endpoint,
	}
}

func NewAgentSSHDialer(endpoint string, user string, signers ...any) (*SSHDialer, error) {
	agentURI := os.Getenv("SSH_AUTH_SOCK")
	if len(signers) == 0 && (agentURI == "" || runtime.GOOS == "windows") {
		return nil, errors.New("no signers provided and no SSH_AUTH_SOCK available")
	}

	var sshSigners []ssh.Signer

	signersProvided := false

	for i, signer := range signers {
		switch castedSigner := signer.(type) {
		case ssh.Signer:
			if i == 0 {
				signersProvided = true
			}
			if !signersProvided {
				return nil, errors.New("multiple signers provided but they aren't all ssh.Signer")
			}
			sshSigners = append(sshSigners, castedSigner)
		case url.URL:
			if signersProvided {
				return nil, errors.New("multiple signers provided but they aren't all ssh.Signer")
			}
			agentURI = castedSigner.String()
		case string:
			if signersProvided {
				return nil, errors.New("multiple signers provided but they aren't all ssh.Signer")
			}
			agentURI = castedSigner
		}
	}

	if !signersProvided {
		var err error
		sshSigners, err = getSignersFromSocket(agentURI)
		if err != nil {
			return nil, err // these are wrapped
		}
	}

	clientConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(sshSigners...)},
	}

	clientConfig.SetDefaults()

	clientConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	return &SSHDialer{
		clientConfig: clientConfig,
		host:         endpoint,
	}, nil
}

func (sshd *SSHDialer) WithHostKeyVerification(callback ssh.HostKeyCallback) *SSHDialer {
	sshd.clientConfig.HostKeyCallback = callback
	return sshd
}

func (sshd *SSHDialer) WithTimeout(timeout time.Duration) *SSHDialer {
	sshd.timeout = timeout
	return sshd
}

func (sshd *SSHDialer) Close() error {
	sshd.mu.Lock()
	defer sshd.mu.Unlock()
	if sshd.clientConn == nil {
		return nil
	}
	err := sshd.clientConn.Close()
	sshd.clientConn = nil
	return err
}

type dialRes struct {
	conn net.Conn
	err  error
}

func (sshd *SSHDialer) dial(resChan chan dialRes, network, addr string) {
	sshd.mu.RLock()
	if sshd.clientConn == nil {
		sshd.mu.RUnlock()
		sshd.mu.Lock()
		var err error
		sshd.clientConn, err = ssh.Dial("tcp", sshd.host, sshd.clientConfig)
		if err != nil {
			if sshd.clientConn != nil {
				ioClose(sshd.clientConn)
			}
			sshd.clientConn = nil
			sshd.mu.Unlock()
			resChan <- dialRes{nil, err}
			return
		}
		sshd.mu.Unlock()
		sshd.mu.RLock()
	}
	sshd.mu.RUnlock()
	c, e := sshd.clientConn.Dial(network, addr)
	resChan <- dialRes{c, e}
}

func (sshd *SSHDialer) DialCtx(ctx context.Context, network, addr string) (net.Conn, error) {
	resChan := make(chan dialRes)
	go sshd.dial(resChan, network, addr)
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	case res := <-resChan:
		return res.conn, res.err
	}
}

func (sshd *SSHDialer) Dial(network, addr string) (net.Conn, error) {
	if sshd.timeout == 0 {
		return sshd.DialCtx(context.Background(), network, addr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), sshd.timeout)
	defer cancel()
	return sshd.DialCtx(ctx, network, addr)
}
