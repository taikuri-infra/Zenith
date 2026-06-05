package hetzner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient creates a client pointed at the given test server URL.
func newTestClient(serverURL, token string) *Client {
	return &Client{
		token:        token,
		baseURL:      serverURL,
		httpClient:   http.DefaultClient,
		pollInterval: time.Millisecond,
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("test-token")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("expected default base URL, got %s", c.baseURL)
	}
}

func TestCreateServer(t *testing.T) {
	var captured CreateServerRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/servers" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer auth, got %s", r.Header.Get("Authorization"))
		}
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(CreateServerResponse{
			Server: Server{ID: 42, Name: captured.Name, Status: "initializing"},
			Action: Action{ID: 1, Command: "create_server", Status: "running"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "test-token")
	resp, err := c.CreateServer(context.Background(), CreateServerRequest{
		Name:       "zenith-mc",
		ServerType: "cx22",
		Image:      "ubuntu-22.04",
		Location:   "fsn1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Server.ID != 42 {
		t.Errorf("expected server ID 42, got %d", resp.Server.ID)
	}
	if captured.ServerType != "cx22" {
		t.Errorf("expected server type cx22, got %s", captured.ServerType)
	}
}

func TestGetServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/99" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"server": Server{ID: 99, Name: "my-server", Status: "running",
				PublicNet: PublicNet{IPv4: IPv4{IP: "10.0.0.1"}}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	server, err := c.GetServer(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.ID != 99 {
		t.Errorf("expected ID 99, got %d", server.ID)
	}
	if server.PublicNet.IPv4.IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", server.PublicNet.IPv4.IP)
	}
}

func TestDeleteServer(t *testing.T) {
	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/servers/1" {
			deleted = true
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	err := c.DeleteServer(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected DELETE to be called")
	}
}

func TestWaitForServerRunning_AlreadyRunning(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]interface{}{
			"server": Server{ID: 42, Name: "test", Status: "running",
				PublicNet: PublicNet{IPv4: IPv4{IP: "1.2.3.4"}}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server, err := c.WaitForServerRunning(ctx, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.Status != "running" {
		t.Errorf("expected running, got %s", server.Status)
	}
	if calls != 1 {
		t.Errorf("expected 1 API call, got %d", calls)
	}
}

func TestWaitForServerRunning_EventuallyRunning(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := "initializing"
		if calls >= 3 {
			status = "running"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"server": Server{ID: 1, Name: "test", Status: status,
				PublicNet: PublicNet{IPv4: IPv4{IP: "5.6.7.8"}}},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server, err := c.WaitForServerRunning(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server.Status != "running" {
		t.Errorf("expected running, got %s", server.Status)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 calls, got %d", calls)
	}
}

func TestWaitForServerRunning_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"server": Server{ID: 1, Name: "test", Status: "initializing"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := c.WaitForServerRunning(ctx, 1)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestAPIError_Returned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "invalid_input",
				"message": "server name already exists",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	_, err := c.GetServer(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("expected 422 in error, got: %v", err)
	}
}

func TestCreateSSHKey(t *testing.T) {
	var captured CreateSSHKeyRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/ssh_keys" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ssh_key": SSHKey{ID: 5, Name: captured.Name, PublicKey: captured.PublicKey},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	key, err := c.CreateSSHKey(context.Background(), CreateSSHKeyRequest{
		Name:      "zenith-mc-key",
		PublicKey: "ssh-rsa AAAA...",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.ID != 5 {
		t.Errorf("expected key ID 5, got %d", key.ID)
	}
	if captured.Name != "zenith-mc-key" {
		t.Errorf("expected key name 'zenith-mc-key', got %q", captured.Name)
	}
}

func TestDeleteSSHKey(t *testing.T) {
	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/ssh_keys/5" {
			deleted = true
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	err := c.DeleteSSHKey(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected DELETE to be called")
	}
}

func TestCreateServer_WithSSHKeys(t *testing.T) {
	var captured CreateServerRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(CreateServerResponse{
			Server: Server{ID: 10, Name: captured.Name, Status: "initializing"},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, "tok")
	_, err := c.CreateServer(context.Background(), CreateServerRequest{
		Name:       "test",
		ServerType: "cx22",
		Image:      "ubuntu-22.04",
		Location:   "fsn1",
		SSHKeys:    []string{"5"},
		Labels:     map[string]string{"managed-by": "zenith"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(captured.SSHKeys) != 1 || captured.SSHKeys[0] != "5" {
		t.Errorf("expected SSHKeys [5], got %v", captured.SSHKeys)
	}
	if captured.Labels["managed-by"] != "zenith" {
		t.Errorf("expected label managed-by=zenith, got %v", captured.Labels)
	}
}
