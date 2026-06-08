package sshclient

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startTestSSHServer starts a minimal SSH server for testing.
// Returns the address and a shutdown function.
func startTestSSHServer(t *testing.T, handler func(ch ssh.Channel, reqs <-chan *ssh.Request)) (string, func()) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostKey, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				sc, chans, reqs, err := ssh.NewServerConn(c, config)
				if err != nil {
					return
				}
				defer sc.Close()
				go ssh.DiscardRequests(reqs)
				for newChan := range chans {
					if newChan.ChannelType() != "session" {
						newChan.Reject(ssh.UnknownChannelType, "unsupported")
						continue
					}
					ch, reqCh, err := newChan.Accept()
					if err != nil {
						continue
					}
					go handler(ch, reqCh)
				}
			}(conn)
		}
	}()

	addr := listener.Addr().String()
	return addr, func() { listener.Close() }
}

func TestDial_NoAuthMethod(t *testing.T) {
	_, err := Dial(Config{
		Host: "127.0.0.1",
		Port: 22,
		User: "root",
		// no PrivateKey, no Password
	})
	if err == nil {
		t.Error("expected error for no auth method")
	}
}

func TestDialWithRetry_FailsAllRetries(t *testing.T) {
	start := time.Now()
	_, err := DialWithRetry(Config{
		Host:     "127.0.0.1",
		Port:     19998,
		User:     "root",
		Password: "x",
		Timeout:  50 * time.Millisecond,
	}, 3, 10*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Error("expected error after retries")
	}
	if elapsed < 20*time.Millisecond {
		t.Errorf("expected at least 20ms for retries, got %v", elapsed)
	}
}

func TestRun(t *testing.T) {
	addr, shutdown := startTestSSHServer(t, func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		for req := range reqs {
			switch req.Type {
			case "exec":
				req.Reply(true, nil)
				ch.Write([]byte("hello from server\n"))
				ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
				ch.CloseWrite() // signal EOF on write side so client reads complete
				return
			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	})
	defer shutdown()

	host, portStr, _ := net.SplitHostPort(addr)
	port := 22
	fmt.Sscanf(portStr, "%d", &port)

	c, err := Dial(Config{
		Host:     host,
		Port:     port,
		User:     "root",
		Password: "any",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	out, err := c.Run("echo hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestRunIgnoreError(t *testing.T) {
	addr, shutdown := startTestSSHServer(t, func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		for req := range reqs {
			if req.Type == "exec" {
				req.Reply(true, nil)
				ch.Write([]byte("some output\n"))
				ch.SendRequest("exit-status", false, []byte{0, 0, 0, 1})
				ch.CloseWrite() // signal EOF so client stdout goroutine exits cleanly
				return
			}
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	})
	defer shutdown()

	host, portStr, _ := net.SplitHostPort(addr)
	port := 22
	fmt.Sscanf(portStr, "%d", &port)

	c, err := Dial(Config{
		Host:     host,
		Port:     port,
		User:     "root",
		Password: "any",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	out := c.RunIgnoreError("exit 1")
	// Should return output even though command failed
	_ = out
}

func TestUpload(t *testing.T) {
	addr, shutdown := startTestSSHServer(t, func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		for req := range reqs {
			if req.Type == "exec" {
				// Reply immediately so client starts sending stdin data.
				req.Reply(true, nil)
				buf := make([]byte, 1024)
				ch.Read(buf) //nolint — we just need stdin to flow; ignore read result
				ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
				return
			}
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	})
	defer shutdown()

	host, portStr, _ := net.SplitHostPort(addr)
	port := 22
	fmt.Sscanf(portStr, "%d", &port)

	c, err := Dial(Config{
		Host:     host,
		Port:     port,
		User:     "root",
		Password: "any",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	testData := []byte("hello world\n")
	err = c.Upload("/tmp/test.txt", testData)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{Host: "example.com", User: "root", Password: "x"}
	if cfg.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", cfg.Host)
	}
	if cfg.Port != 0 {
		t.Errorf("expected default port 0 (will be filled in Dial), got %d", cfg.Port)
	}
}

// startTestSSHServerWithHostKey is like startTestSSHServer but also returns the
// server's public host key so tests can use it for TOFU verification.
func startTestSSHServerWithHostKey(t *testing.T, handler func(ch ssh.Channel, reqs <-chan *ssh.Request)) (addr string, hostPubKey ssh.PublicKey, shutdown func()) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostKey, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				sc, chans, reqs, err := ssh.NewServerConn(c, config)
				if err != nil {
					return
				}
				defer sc.Close()
				go ssh.DiscardRequests(reqs)
				for newChan := range chans {
					if newChan.ChannelType() != "session" {
						newChan.Reject(ssh.UnknownChannelType, "unsupported")
						continue
					}
					ch, reqCh, err := newChan.Accept()
					if err != nil {
						continue
					}
					go handler(ch, reqCh)
				}
			}(conn)
		}
	}()

	return listener.Addr().String(), hostKey.PublicKey(), func() { listener.Close() }
}

// execHandler is a minimal SSH session handler that accepts exec requests and
// exits cleanly. It also records the last exec command string into lastCmd.
func makeRecordingExecHandler(lastCmd *string) func(ch ssh.Channel, reqs <-chan *ssh.Request) {
	return func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		for req := range reqs {
			if req.Type == "exec" {
				// Payload: 4-byte length-prefixed string.
				if len(req.Payload) >= 4 {
					cmdLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
					if cmdLen <= len(req.Payload)-4 {
						*lastCmd = string(req.Payload[4 : 4+cmdLen])
					}
				}
				req.Reply(true, nil)
				// Drain stdin so the client's Write doesn't block.
				buf := make([]byte, 4096)
				ch.Read(buf) //nolint
				ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
				return
			}
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func dialTestServer(t *testing.T, addr string, knownHostKey []byte) (*Client, error) {
	t.Helper()
	host, portStr, _ := net.SplitHostPort(addr)
	port := 22
	fmt.Sscanf(portStr, "%d", &port)
	return Dial(Config{
		Host:         host,
		Port:         port,
		User:         "root",
		Password:     "any",
		Timeout:      5 * time.Second,
		KnownHostKey: knownHostKey,
	})
}

func TestDial_TOFU_CapturesHostKey(t *testing.T) {
	addr, _, shutdown := startTestSSHServerWithHostKey(t, func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		ssh.DiscardRequests(reqs)
	})
	defer shutdown()

	c, err := dialTestServer(t, addr, nil) // nil = TOFU mode
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	captured := c.CapturedHostKey()
	if len(captured) == 0 {
		t.Fatal("expected CapturedHostKey to be non-nil after TOFU connect, got empty")
	}
}

func TestDial_KnownHostKey_Correct(t *testing.T) {
	addr, serverPubKey, shutdown := startTestSSHServerWithHostKey(t, func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		ssh.DiscardRequests(reqs)
	})
	defer shutdown()

	// First connect: TOFU — capture the key.
	c1, err := dialTestServer(t, addr, nil)
	if err != nil {
		t.Fatalf("first dial: %v", err)
	}
	captured := c1.CapturedHostKey()
	c1.Close()

	// Sanity check: captured key should match the server's actual public key.
	if string(captured) != string(serverPubKey.Marshal()) {
		t.Fatal("captured key does not match server public key")
	}

	// Second connect: verify mode — should succeed with the correct key.
	c2, err := dialTestServer(t, addr, captured)
	if err != nil {
		t.Fatalf("second dial with correct key: %v", err)
	}
	defer c2.Close()
}

func TestDial_KnownHostKey_Mismatch(t *testing.T) {
	addr, _, shutdown := startTestSSHServerWithHostKey(t, func(ch ssh.Channel, reqs <-chan *ssh.Request) {
		defer ch.Close()
		ssh.DiscardRequests(reqs)
	})
	defer shutdown()

	// Generate a different RSA key and marshal it as a "wrong" known host key.
	wrongPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}
	wrongPubKey, err := ssh.NewPublicKey(&wrongPrivKey.PublicKey)
	if err != nil {
		t.Fatalf("new public key: %v", err)
	}
	wrongKnownKey := wrongPubKey.Marshal()

	_, err = dialTestServer(t, addr, wrongKnownKey)
	if err == nil {
		t.Fatal("expected dial to fail with mismatched host key, but it succeeded")
	}
}

func TestUpload_PathWithSpaces(t *testing.T) {
	var lastCmd string
	addr, _, shutdown := startTestSSHServerWithHostKey(t, makeRecordingExecHandler(&lastCmd))
	defer shutdown()

	c, err := dialTestServer(t, addr, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	remotePath := "/tmp/my file with spaces.txt"
	err = c.Upload(remotePath, []byte("data"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	// The command sent to the server must use single-quote wrapping.
	expected := "cat > '/tmp/my file with spaces.txt'"
	if lastCmd != expected {
		t.Errorf("expected command %q, got %q", expected, lastCmd)
	}
}
