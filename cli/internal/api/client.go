package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the Zenith API client used by the CLI.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// Project operations

type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Owner       string `json:"owner"`
	Plan        string `json:"plan"`
	Region      string `json:"region"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func (c *Client) ListProjects() ([]Project, error) {
	var resp struct {
		Items []Project `json:"items"`
	}
	if err := c.do("GET", "/api/v1/projects", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) GetProject(id string) (*Project, error) {
	var p Project
	if err := c.do("GET", "/api/v1/projects/"+id, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// App operations

type App struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas int    `json:"replicas"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
	CPU      string `json:"cpu"`
	Memory   string `json:"memory"`
}

func (c *Client) ListApps(projectID string) ([]App, error) {
	var resp struct {
		Items []App `json:"items"`
	}
	if err := c.do("GET", fmt.Sprintf("/api/v1/projects/%s/apps", projectID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) CreateApp(projectID string, app *App) (*App, error) {
	var result App
	if err := c.do("POST", fmt.Sprintf("/api/v1/projects/%s/apps", projectID), app, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Redeploy(projectID, appName string) error {
	return c.do("POST", fmt.Sprintf("/api/v1/projects/%s/apps/%s/redeploy", projectID, appName), nil, nil)
}

// Database operations

type Database struct {
	Name             string `json:"name"`
	Engine           string `json:"engine"`
	Version          string `json:"version"`
	Storage          string `json:"storage"`
	Status           string `json:"status"`
	ConnectionString string `json:"connection_string"`
	Port             int    `json:"port"`
}

func (c *Client) ListDatabases(projectID string) ([]Database, error) {
	var resp struct {
		Items []Database `json:"items"`
	}
	if err := c.do("GET", fmt.Sprintf("/api/v1/projects/%s/databases", projectID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) CreateDatabase(projectID string, db *Database) (*Database, error) {
	var result Database
	if err := c.do("POST", fmt.Sprintf("/api/v1/projects/%s/databases", projectID), db, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDatabase(projectID, name string) (*Database, error) {
	var db Database
	if err := c.do("GET", fmt.Sprintf("/api/v1/projects/%s/databases/%s", projectID, name), nil, &db); err != nil {
		return nil, err
	}
	return &db, nil
}
