package sshclient

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH connection.
type Client struct {
	inner *ssh.Client
}

// Config holds SSH connection parameters.
type Config struct {
	Host       string
	Port       int
	User       string
	PrivateKey []byte // PEM-encoded private key; mutually exclusive with Password
	Password   string
	Timeout    time.Duration
}

// Dial connects to an SSH server and returns a Client.
func Dial(cfg Config) (*Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	var authMethods []ssh.AuthMethod
	if len(cfg.PrivateKey) > 0 {
		signer, err := ssh.ParsePrivateKey(cfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no auth method provided (need PrivateKey or Password)")
	}

	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec — bootstrap installer; we just created the server
		Timeout:         cfg.Timeout,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return &Client{inner: conn}, nil
}

// Run executes a command and returns combined stdout+stderr output.
func (c *Client) Run(cmd string) (string, error) {
	sess, err := c.inner.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var out bytes.Buffer
	sess.Stdout = &out
	sess.Stderr = &out

	if err := sess.Run(cmd); err != nil {
		return out.String(), fmt.Errorf("run %q: %w", cmd, err)
	}
	return out.String(), nil
}

// RunIgnoreError executes a command and returns output even on non-zero exit.
func (c *Client) RunIgnoreError(cmd string) string {
	out, _ := c.Run(cmd)
	return out
}

// Upload writes data to a remote file path via stdin redirection.
func (c *Client) Upload(remotePath string, data []byte) error {
	sess, err := c.inner.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	sess.Stdin = bytes.NewReader(data)
	if err := sess.Run(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("upload to %s: %w", remotePath, err)
	}
	return nil
}

// Close closes the underlying SSH connection.
func (c *Client) Close() error {
	return c.inner.Close()
}

// DialWithRetry attempts Dial up to maxRetries times with retryInterval between attempts.
func DialWithRetry(cfg Config, maxRetries int, retryInterval time.Duration) (*Client, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		c, err := Dial(cfg)
		if err == nil {
			return c, nil
		}
		lastErr = err
		time.Sleep(retryInterval)
	}
	return nil, fmt.Errorf("ssh dial failed after %d attempts: %w", maxRetries, lastErr)
}
