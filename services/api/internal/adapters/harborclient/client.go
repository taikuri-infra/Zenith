package harborclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides access to the Harbor v2.0 REST API.
type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

// New creates a Harbor API client.
func New(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

// Repository represents a Harbor repository.
type Repository struct {
	Name          string `json:"name"`
	ArtifactCount int    `json:"artifact_count"`
	UpdateTime    string `json:"update_time"`
}

// Artifact represents a single image artifact (tag).
type Artifact struct {
	Digest    string    `json:"digest"`
	Size      int64     `json:"size"`
	PushTime  time.Time `json:"push_time"`
	Tags      []Tag     `json:"tags"`
	ScanOverview map[string]*ScanOverview `json:"scan_overview,omitempty"`
}

// Tag holds tag metadata.
type Tag struct {
	Name     string    `json:"name"`
	PushTime time.Time `json:"push_time"`
}

// ScanOverview holds Trivy scan results for an artifact.
type ScanOverview struct {
	ScanStatus string    `json:"scan_status"` // "Success", "Error", "Running", "Pending"
	Severity   string    `json:"severity"`    // "Critical", "High", "Medium", "Low", "None"
	Summary    *VulnSummary `json:"summary,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

// VulnSummary holds vulnerability count by severity.
type VulnSummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
}

// ListRepositories lists all repositories in a Harbor project.
func (c *Client) ListRepositories(ctx context.Context, project string) ([]Repository, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories?page_size=100", c.baseURL, project)
	var repos []Repository
	if err := c.get(ctx, url, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// ListArtifacts lists all artifacts (tags) for a repository. The withScanOverview
// parameter requests Harbor to include Trivy scan results.
func (c *Client) ListArtifacts(ctx context.Context, project, repo string, withScanOverview bool) ([]Artifact, error) {
	url := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts?page_size=50", c.baseURL, project, repo)
	if withScanOverview {
		url += "&with_scan_overview=true&with_tag=true"
	}
	var artifacts []Artifact
	if err := c.get(ctx, url, &artifacts); err != nil {
		return nil, err
	}
	return artifacts, nil
}

// CreateProject creates a new Harbor project with the given name and optional storage quota in bytes.
// If storageQuota is <= 0, no quota is applied (unlimited).
func (c *Client) CreateProject(ctx context.Context, projectName string, storageQuota int64) error {
	body := map[string]interface{}{
		"project_name": projectName,
		"public":       false,
	}
	if storageQuota > 0 {
		body["storage_limit"] = storageQuota
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v2.0/projects", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("harbor create project failed: %w", err)
	}
	defer resp.Body.Close()

	// 201 = created, 409 = already exists (idempotent)
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("harbor create project error %d: %s", resp.StatusCode, string(respBody))
}

func (c *Client) get(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("harbor request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("harbor API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
