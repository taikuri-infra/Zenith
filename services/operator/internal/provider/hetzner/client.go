package hetzner

import (
	"context"
	"fmt"
)

// Client wraps Hetzner Cloud API operations needed by the operator.
// In production this uses hcloud-go; here we define the interface for testability.
type Client struct {
	token string
}

func NewClient(token string) *Client {
	return &Client{token: token}
}

// Volume operations

type Volume struct {
	ID       int64
	Name     string
	SizeGB   int
	Location string
	Status   string
}

func (c *Client) CreateVolume(ctx context.Context, name string, sizeGB int, location string) (*Volume, error) {
	if c.token == "" {
		return nil, fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with hcloud-go
	return &Volume{
		Name:     name,
		SizeGB:   sizeGB,
		Location: location,
		Status:   "available",
	}, nil
}

func (c *Client) DeleteVolume(ctx context.Context, id int64) error {
	if c.token == "" {
		return fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with hcloud-go
	return nil
}

func (c *Client) ResizeVolume(ctx context.Context, id int64, newSizeGB int) error {
	if c.token == "" {
		return fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with hcloud-go
	return nil
}

// Server operations

type Server struct {
	ID         int64
	Name       string
	ServerType string
	Status     string
	PublicIPv4 string
	PrivateIP  string
}

func (c *Client) CreateServer(ctx context.Context, name, serverType, image, location string) (*Server, error) {
	if c.token == "" {
		return nil, fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with hcloud-go
	return &Server{
		Name:       name,
		ServerType: serverType,
		Status:     "running",
	}, nil
}

func (c *Client) DeleteServer(ctx context.Context, id int64) error {
	if c.token == "" {
		return fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with hcloud-go
	return nil
}

// DNS operations

type DNSRecord struct {
	ID     string
	Type   string
	Name   string
	Value  string
	TTL    int
	ZoneID string
}

func (c *Client) CreateDNSRecord(ctx context.Context, zoneID, recordType, name, value string, ttl int) (*DNSRecord, error) {
	if c.token == "" {
		return nil, fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with Hetzner DNS API
	return &DNSRecord{
		Type:   recordType,
		Name:   name,
		Value:  value,
		TTL:    ttl,
		ZoneID: zoneID,
	}, nil
}

func (c *Client) DeleteDNSRecord(ctx context.Context, recordID string) error {
	if c.token == "" {
		return fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with Hetzner DNS API
	return nil
}

// Object Storage operations

type ObjectStoreBucket struct {
	Name     string
	Endpoint string
	Region   string
}

func (c *Client) CreateBucket(ctx context.Context, name, region string) (*ObjectStoreBucket, error) {
	if c.token == "" {
		return nil, fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with Hetzner S3 API
	return &ObjectStoreBucket{
		Name:     name,
		Region:   region,
		Endpoint: fmt.Sprintf("https://%s.%s.your-objectstorage.com", name, region),
	}, nil
}

func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	if c.token == "" {
		return fmt.Errorf("hetzner token not configured")
	}
	// TODO: implement with Hetzner S3 API
	return nil
}

// IsConfigured checks if the client has a valid token
func (c *Client) IsConfigured() bool {
	return c.token != ""
}
