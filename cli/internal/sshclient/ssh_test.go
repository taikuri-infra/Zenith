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
