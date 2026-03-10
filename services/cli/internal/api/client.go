package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dotechhq/zenith/services/cli/internal/config"
)

// Client wraps the Zenith API for CLI use.
type Client struct {
	cfg  *config.Config
	http *http.Client
}

// New creates an API client from the current config.
func New(cfg *config.Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// do executes an authenticated API request.
func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.cfg.APIBaseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &apiErr)
		msg := apiErr.Error
		if msg == "" {
			msg = apiErr.Message
		}
		if msg == "" {
			msg = string(respBody)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}

// Get performs an authenticated GET request.
func (c *Client) Get(path string, result interface{}) error {
	return c.do("GET", path, nil, result)
}

// Post performs an authenticated POST request.
func (c *Client) Post(path string, body, result interface{}) error {
	return c.do("POST", path, body, result)
}

// Put performs an authenticated PUT request.
func (c *Client) Put(path string, body, result interface{}) error {
	return c.do("PUT", path, body, result)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(path string) error {
	return c.do("DELETE", path, nil, nil)
}

// Login authenticates and stores the tokens.
func (c *Client) Login(email, password string) error {
	var result struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	err := c.Post("/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, &result)
	if err != nil {
		return err
	}

	c.cfg.AccessToken = result.Token
	c.cfg.RefreshToken = result.RefreshToken
	return c.cfg.Save()
}
