package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		// Deploy tokens are stored as "znt_id_xxx:znt_sk_xxx" — use DeployToken scheme.
		// Regular JWT tokens use Bearer.
		if strings.HasPrefix(c.Token, "znt_id_") {
			req.Header.Set("Authorization", "DeployToken "+c.Token)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
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
	ID           string `json:"id"`
	Name         string `json:"name"`
	Image        string `json:"image_url"`
	Replicas     int    `json:"replicas"`
	Port         int    `json:"port"`
	Status       string `json:"status"`
	CPU          string `json:"cpu"`
	Memory       string `json:"memory"`
	URL          string `json:"url"`
	ProjectID    string `json:"project_id"`
	DeploySource string `json:"deploy_source"`
	AppType      string `json:"app_type"`
}

// ListApps returns all apps for a project (or all user apps if projectID is empty).
func (c *Client) ListApps(projectID string) ([]App, error) {
	path := "/api/v1/apps"
	if projectID != "" {
		path = fmt.Sprintf("/api/v1/projects/%s/apps", projectID)
	}
	var resp struct {
		Items []App `json:"items"`
	}
	if err := c.do("GET", path, nil, &resp); err != nil {
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

// DeployResult is the response from POST /api/v1/deploy (CI/CD deploy endpoint).
type DeployResult struct {
	DeploymentID string `json:"deployment_id"`
	AppID        string `json:"app_id"`
	AppName      string `json:"app_name"`
	Status       string `json:"status"`
	URL          string `json:"url"`
}

// DeploymentStatus is the response from GET /api/v1/deployments/:id.
type DeploymentStatus struct {
	ID     string `json:"id"`
	AppID  string `json:"app_id"`
	Status string `json:"status"`
	Image  string `json:"image"`
	URL    string `json:"url"`
}

// Deploy triggers an image deploy via the CI/CD endpoint.
// Requires a deploy token set as the client's Token.
func (c *Client) Deploy(appName, image, environment string, replicas int) (*DeployResult, error) {
	payload := map[string]interface{}{
		"app":         appName,
		"image":       image,
		"environment": environment,
	}
	if replicas > 0 {
		payload["replicas"] = replicas
	}
	var result DeployResult
	if err := c.do("POST", "/api/v1/deploy", payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDeploymentStatus returns the current status of a deployment.
func (c *Client) GetDeploymentStatus(deploymentID string) (*DeploymentStatus, error) {
	var result DeploymentStatus
	if err := c.do("GET", "/api/v1/deployments/"+deploymentID, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
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

// Log operations

// LogEntry is a single log line from the monitoring API.
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Line      string            `json:"line"`
	Level     string            `json:"level"`
	Labels    map[string]string `json:"labels"`
}

// LogsResponse is the response from GET /api/v1/apps/:id/logs.
type LogsResponse struct {
	Entries []LogEntry `json:"entries"`
	Total   int        `json:"total"`
}

// GetAppLogs fetches historical logs for an app (non-streaming).
func (c *Client) GetAppLogs(appID, level, since string, limit int) (*LogsResponse, error) {
	path := fmt.Sprintf("/api/v1/apps/%s/logs?limit=%d", appID, limit)
	if level != "" {
		path += "&level=" + level
	}
	if since != "" {
		path += "&since=" + since
	}
	var result LogsResponse
	if err := c.do("GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StreamAppLogs connects to the SSE log stream and calls handler for each entry.
// Returns when the stream ends or handler returns false.
// Uses a separate HTTP client with no timeout for long-lived connections.
func (c *Client) StreamAppLogs(appID string, handler func(entry LogEntry) bool) error {
	req, err := http.NewRequest("GET", c.BaseURL+fmt.Sprintf("/api/v1/apps/%s/logs/stream", appID), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if c.Token != "" {
		if strings.HasPrefix(c.Token, "znt_id_") {
			req.Header.Set("Authorization", "DeployToken "+c.Token)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}
	req.Header.Set("Accept", "text/event-stream")

	// Use a no-timeout client for the streaming connection.
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to log stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[6:]
		if data == "{}" {
			break // done event
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			continue
		}
		if !handler(entry) {
			break
		}
	}
	return scanner.Err()
}

// DoRaw exposes the internal do method for arbitrary API calls.
func (c *Client) DoRaw(method, path string, body interface{}, result interface{}) error {
	return c.do(method, path, body, result)
}
