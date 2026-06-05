package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.cloudflare.com/client/v4"

// Client is a minimal Cloudflare API client for DNS management.
type Client struct {
	token      string
	apiBase    string
	httpClient *http.Client
}

// NewClient creates a Cloudflare client with the given API token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		apiBase:    defaultAPIBase,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Zone represents a Cloudflare DNS zone.
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DNSRecord represents a Cloudflare DNS record.
type DNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type cfResponse struct {
	Success bool            `json:"success"`
	Errors  []cfError       `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
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

	req, err := http.NewRequest(method, c.apiBase+path, reqBody)
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

	var cf cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cf); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if !cf.Success {
		msgs := make([]string, len(cf.Errors))
		for i, e := range cf.Errors {
			msgs[i] = fmt.Sprintf("%d: %s", e.Code, e.Message)
		}
		return fmt.Errorf("cloudflare API error: %s", strings.Join(msgs, "; "))
	}
	if result != nil {
		return json.Unmarshal(cf.Result, result)
	}
	return nil
}

// FindZone returns the zone for the given domain (strips to apex domain).
func (c *Client) FindZone(domain string) (*Zone, error) {
	// Strip to apex domain (last two labels)
	parts := strings.Split(domain, ".")
	apex := domain
	if len(parts) > 2 {
		apex = strings.Join(parts[len(parts)-2:], ".")
	}

	var zones []Zone
	if err := c.do("GET", "/zones?name="+apex, nil, &zones); err != nil {
		return nil, err
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no Cloudflare zone found for domain %q", apex)
	}
	return &zones[0], nil
}

// CreateRecord creates a DNS A record in the given zone.
func (c *Client) CreateRecord(zoneID, name, ip string) (*DNSRecord, error) {
	payload := map[string]interface{}{
		"type":    "A",
		"name":    name,
		"content": ip,
		"ttl":     120,
		"proxied": false,
	}
	var record DNSRecord
	if err := c.do("POST", fmt.Sprintf("/zones/%s/dns_records", zoneID), payload, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpsertRecord creates or updates a DNS A record for the given subdomain.
func (c *Client) UpsertRecord(zoneID, name, ip string) error {
	var existing []DNSRecord
	if err := c.do("GET", fmt.Sprintf("/zones/%s/dns_records?type=A&name=%s", zoneID, name), nil, &existing); err != nil {
		return err
	}

	if len(existing) > 0 {
		payload := map[string]interface{}{
			"type":    "A",
			"name":    name,
			"content": ip,
			"ttl":     120,
			"proxied": false,
		}
		return c.do("PUT", fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, existing[0].ID), payload, nil)
	}

	_, err := c.CreateRecord(zoneID, name, ip)
	return err
}
