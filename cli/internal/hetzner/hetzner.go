package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.hetzner.cloud/v1"

// Client is a minimal Hetzner Cloud API client.
type Client struct {
	token        string
	baseURL      string
	httpClient   *http.Client
	pollInterval time.Duration
}

// NewClient creates a Hetzner API client with the given token.
func NewClient(token string) *Client {
	return &Client{
		token:        token,
		baseURL:      defaultBaseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		pollInterval: 3 * time.Second,
	}
}

// Server represents a Hetzner Cloud server.
type Server struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	PublicNet PublicNet `json:"public_net"`
	Created   time.Time `json:"created"`
}

// PublicNet holds a server's public network configuration.
type PublicNet struct {
	IPv4 IPv4 `json:"ipv4"`
}

// IPv4 holds the public IPv4 address.
type IPv4 struct {
	IP string `json:"ip"`
}

// CreateServerRequest is the payload for server creation.
type CreateServerRequest struct {
	Name       string            `json:"name"`
	ServerType string            `json:"server_type"`
	Image      string            `json:"image"`
	Location   string            `json:"location"`
	SSHKeys    []string          `json:"ssh_keys,omitempty"`
	UserData   string            `json:"user_data,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// CreateServerResponse is returned by server creation.
type CreateServerResponse struct {
	Server Server `json:"server"`
	Action Action `json:"action"`
}

// Action represents a Hetzner async operation.
type Action struct {
	ID       int64  `json:"id"`
	Command  string `json:"command"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
}

// SSHKey represents a Hetzner SSH key resource.
type SSHKey struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

// CreateSSHKeyRequest is the payload for SSH key creation.
type CreateSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("hetzner API %d: %s — %s", resp.StatusCode, errResp.Error.Code, errResp.Error.Message)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// CreateServer creates a new server and returns the response.
func (c *Client) CreateServer(ctx context.Context, req CreateServerRequest) (*CreateServerResponse, error) {
	var resp CreateServerResponse
	if err := c.do(ctx, "POST", "/servers", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetServer retrieves a server by ID.
func (c *Client) GetServer(ctx context.Context, id int64) (*Server, error) {
	var resp struct {
		Server Server `json:"server"`
	}
	if err := c.do(ctx, "GET", fmt.Sprintf("/servers/%d", id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Server, nil
}

// DeleteServer deletes a server by ID.
func (c *Client) DeleteServer(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/servers/%d", id), nil, nil)
}

// WaitForServerRunning polls until the server status is "running" or ctx is cancelled.
func (c *Client) WaitForServerRunning(ctx context.Context, id int64) (*Server, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		srv, err := c.GetServer(ctx, id)
		if err != nil {
			return nil, err
		}
		if srv.Status == "running" {
			return srv, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(c.pollInterval):
		}
	}
}

// CreateSSHKey uploads a public key to Hetzner and returns the key resource.
func (c *Client) CreateSSHKey(ctx context.Context, req CreateSSHKeyRequest) (*SSHKey, error) {
	var resp struct {
		SSHKey SSHKey `json:"ssh_key"`
	}
	if err := c.do(ctx, "POST", "/ssh_keys", req, &resp); err != nil {
		return nil, err
	}
	return &resp.SSHKey, nil
}

// DeleteSSHKey removes an SSH key by ID.
func (c *Client) DeleteSSHKey(ctx context.Context, id int64) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/ssh_keys/%d", id), nil, nil)
}
