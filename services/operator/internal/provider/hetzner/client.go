package hetzner

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Client wraps Hetzner Cloud API operations needed by the operator.
type Client struct {
	token  string
	hcloud *hcloud.Client
}

func NewClient(token string) *Client {
	c := &Client{token: token}
	if token != "" {
		c.hcloud = hcloud.NewClient(hcloud.WithToken(token))
	}
	return c
}

// IsConfigured checks if the client has a valid token
func (c *Client) IsConfigured() bool {
	return c.token != "" && c.hcloud != nil
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
	if !c.IsConfigured() {
		return nil, fmt.Errorf("hetzner token not configured")
	}

	loc, _, err := c.hcloud.Location.GetByName(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("get location %s: %w", location, err)
	}
	if loc == nil {
		return nil, fmt.Errorf("location %s not found", location)
	}

	result, _, err := c.hcloud.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Name:     name,
		Size:     sizeGB,
		Location: loc,
		Labels: map[string]string{
			"managed-by": "zenith-operator",
		},
		Format: hcloud.Ptr("ext4"),
	})
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	return &Volume{
		ID:       result.Volume.ID,
		Name:     result.Volume.Name,
		SizeGB:   result.Volume.Size,
		Location: result.Volume.Location.Name,
		Status:   string(result.Volume.Status),
	}, nil
}

func (c *Client) GetVolume(ctx context.Context, id int64) (*Volume, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("hetzner token not configured")
	}

	vol, _, err := c.hcloud.Volume.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get volume %d: %w", id, err)
	}
	if vol == nil {
		return nil, fmt.Errorf("volume %d not found", id)
	}

	return &Volume{
		ID:       vol.ID,
		Name:     vol.Name,
		SizeGB:   vol.Size,
		Location: vol.Location.Name,
		Status:   string(vol.Status),
	}, nil
}

func (c *Client) DeleteVolume(ctx context.Context, id int64) error {
	if !c.IsConfigured() {
		return fmt.Errorf("hetzner token not configured")
	}

	vol := &hcloud.Volume{ID: id}
	// Detach first if attached
	if _, _, err := c.hcloud.Volume.Detach(ctx, vol); err != nil {
		// Ignore detach errors (may not be attached)
	}

	_, err := c.hcloud.Volume.Delete(ctx, vol)
	if err != nil {
		return fmt.Errorf("delete volume %d: %w", id, err)
	}
	return nil
}

func (c *Client) ResizeVolume(ctx context.Context, id int64, newSizeGB int) error {
	if !c.IsConfigured() {
		return fmt.Errorf("hetzner token not configured")
	}

	vol := &hcloud.Volume{ID: id}
	_, _, err := c.hcloud.Volume.Resize(ctx, vol, newSizeGB)
	if err != nil {
		return fmt.Errorf("resize volume %d to %dGB: %w", id, newSizeGB, err)
	}
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
	if !c.IsConfigured() {
		return nil, fmt.Errorf("hetzner token not configured")
	}

	st, _, err := c.hcloud.ServerType.GetByName(ctx, serverType)
	if err != nil || st == nil {
		return nil, fmt.Errorf("server type %s not found", serverType)
	}

	img, _, err := c.hcloud.Image.GetByNameAndArchitecture(ctx, image, hcloud.ArchitectureX86)
	if err != nil || img == nil {
		return nil, fmt.Errorf("image %s not found", image)
	}

	loc, _, err := c.hcloud.Location.GetByName(ctx, location)
	if err != nil || loc == nil {
		return nil, fmt.Errorf("location %s not found", location)
	}

	result, _, err := c.hcloud.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: st,
		Image:      img,
		Location:   loc,
		Labels: map[string]string{
			"managed-by": "zenith-operator",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}

	srv := result.Server
	publicIP := ""
	if !srv.PublicNet.IPv4.IP.IsUnspecified() {
		publicIP = srv.PublicNet.IPv4.IP.String()
	}

	return &Server{
		ID:         srv.ID,
		Name:       srv.Name,
		ServerType: srv.ServerType.Name,
		Status:     string(srv.Status),
		PublicIPv4: publicIP,
	}, nil
}

func (c *Client) DeleteServer(ctx context.Context, id int64) error {
	if !c.IsConfigured() {
		return fmt.Errorf("hetzner token not configured")
	}

	srv := &hcloud.Server{ID: id}
	_, _, err := c.hcloud.Server.DeleteWithResult(ctx, srv)
	if err != nil {
		return fmt.Errorf("delete server %d: %w", id, err)
	}
	return nil
}

// DNS operations (Hetzner DNS API - separate from hcloud-go)
// Note: Hetzner DNS is a separate API, not part of hcloud-go.
// These use HTTP calls to https://dns.hetzner.com/api/v1/

type DNSRecord struct {
	ID     string
	Type   string
	Name   string
	Value  string
	TTL    int
	ZoneID string
}

func (c *Client) CreateDNSRecord(ctx context.Context, zoneID, recordType, name, value string, ttl int) (*DNSRecord, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("hetzner token not configured")
	}

	// Hetzner DNS API is separate from hcloud-go
	// In production: POST https://dns.hetzner.com/api/v1/records
	// with Authorization: Bearer {token}
	return &DNSRecord{
		ID:     fmt.Sprintf("dns-%s-%s", zoneID, name),
		Type:   recordType,
		Name:   name,
		Value:  value,
		TTL:    ttl,
		ZoneID: zoneID,
	}, nil
}

func (c *Client) DeleteDNSRecord(ctx context.Context, recordID string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("hetzner token not configured")
	}
	// In production: DELETE https://dns.hetzner.com/api/v1/records/{recordID}
	return nil
}

// Object Storage operations (Hetzner S3-compatible API)

type ObjectStoreBucket struct {
	Name      string
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
}

func (c *Client) CreateBucket(ctx context.Context, name, region string) (*ObjectStoreBucket, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("hetzner token not configured")
	}

	// Hetzner Object Storage is S3-compatible
	// In production: use AWS SDK with Hetzner endpoint
	endpoint := fmt.Sprintf("https://%s.%s.your-objectstorage.com", name, region)

	return &ObjectStoreBucket{
		Name:     name,
		Region:   region,
		Endpoint: endpoint,
	}, nil
}

func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	if !c.IsConfigured() {
		return fmt.Errorf("hetzner token not configured")
	}
	return nil
}

// ParseVolumeID parses a string volume ID to int64.
func ParseVolumeID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}
