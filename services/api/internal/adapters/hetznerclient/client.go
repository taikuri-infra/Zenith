package hetznerclient

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	labelManagedBy = "zenith.dev/managed-by"
	labelRole      = "zenith.dev/role"
	managedValue   = "autoscaler"
	roleWorker     = "worker"
)

// HetznerAPI defines the operations the autoscaler needs from Hetzner Cloud.
type HetznerAPI interface {
	CreateServer(ctx context.Context, name, serverType, location, userData string) (*ServerResult, error)
	DeleteServer(ctx context.Context, serverID int64) error
	ListServers(ctx context.Context) ([]ServerResult, error)
	GetServer(ctx context.Context, serverID int64) (*ServerResult, error)
}

// ServerResult is the information returned about a Hetzner server.
type ServerResult struct {
	ID         int64
	Name       string
	PublicIPv4 string
	Status     string // running, starting, stopping, off
	ServerType string
	CPUCores   int
	RAMMB      int
	MonthlyCost float64
}

// Client wraps the hcloud-go SDK.
type Client struct {
	hc *hcloud.Client
}

// NewClient creates an authenticated Hetzner Cloud client.
func NewClient(token string) *Client {
	return &Client{
		hc: hcloud.NewClient(hcloud.WithToken(token)),
	}
}

// CreateServer provisions a new Hetzner server with the given cloud-init user data.
func (c *Client) CreateServer(ctx context.Context, name, serverType, location, userData string) (*ServerResult, error) {
	st, _, err := c.hc.ServerType.GetByName(ctx, serverType)
	if err != nil {
		return nil, fmt.Errorf("hetzner: server type lookup %q: %w", serverType, err)
	}
	if st == nil {
		return nil, fmt.Errorf("hetzner: server type %q not found", serverType)
	}

	loc, _, err := c.hc.Location.GetByName(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("hetzner: location lookup %q: %w", location, err)
	}
	if loc == nil {
		return nil, fmt.Errorf("hetzner: location %q not found", location)
	}

	img, _, err := c.hc.Image.GetByNameAndArchitecture(ctx, "ubuntu-24.04", hcloud.ArchitectureX86)
	if err != nil {
		return nil, fmt.Errorf("hetzner: image lookup: %w", err)
	}
	if img == nil {
		return nil, fmt.Errorf("hetzner: ubuntu-24.04 image not found")
	}

	opts := hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: st,
		Location:   loc,
		Image:      img,
		UserData:   userData,
		Labels: map[string]string{
			labelManagedBy: managedValue,
			labelRole:      roleWorker,
		},
	}

	result, _, err := c.hc.Server.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("hetzner: create server %q: %w", name, err)
	}

	srv := result.Server
	ip := ""
	if srv.PublicNet.IPv4.IP != nil {
		ip = srv.PublicNet.IPv4.IP.String()
	}

	monthlyCost := parseMonthlyCost(srv.ServerType.Pricings)

	return &ServerResult{
		ID:          srv.ID,
		Name:        srv.Name,
		PublicIPv4:  ip,
		Status:      string(srv.Status),
		ServerType:  srv.ServerType.Name,
		CPUCores:    srv.ServerType.Cores,
		RAMMB:       int(srv.ServerType.Memory * 1024),
		MonthlyCost: monthlyCost,
	}, nil
}

// DeleteServer removes a Hetzner server by ID.
func (c *Client) DeleteServer(ctx context.Context, serverID int64) error {
	srv := &hcloud.Server{ID: serverID}
	_, _, err := c.hc.Server.DeleteWithResult(ctx, srv)
	if err != nil {
		return fmt.Errorf("hetzner: delete server %d: %w", serverID, err)
	}
	return nil
}

// ListServers returns all Hetzner servers managed by the autoscaler.
func (c *Client) ListServers(ctx context.Context) ([]ServerResult, error) {
	opts := hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelManagedBy + "=" + managedValue,
		},
	}

	servers, err := c.hc.Server.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("hetzner: list servers: %w", err)
	}

	results := make([]ServerResult, 0, len(servers))
	for _, srv := range servers {
		ip := ""
		if srv.PublicNet.IPv4.IP != nil {
			ip = srv.PublicNet.IPv4.IP.String()
		}
		results = append(results, ServerResult{
			ID:          srv.ID,
			Name:        srv.Name,
			PublicIPv4:  ip,
			Status:      string(srv.Status),
			ServerType:  srv.ServerType.Name,
			CPUCores:    srv.ServerType.Cores,
			RAMMB:       int(srv.ServerType.Memory * 1024),
			MonthlyCost: parseMonthlyCost(srv.ServerType.Pricings),
		})
	}

	return results, nil
}

// GetServer returns details for a single Hetzner server.
func (c *Client) GetServer(ctx context.Context, serverID int64) (*ServerResult, error) {
	srv, _, err := c.hc.Server.GetByID(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("hetzner: get server %d: %w", serverID, err)
	}
	if srv == nil {
		return nil, fmt.Errorf("hetzner: server %d not found", serverID)
	}

	ip := ""
	if srv.PublicNet.IPv4.IP != nil {
		ip = srv.PublicNet.IPv4.IP.String()
	}

	return &ServerResult{
		ID:          srv.ID,
		Name:        srv.Name,
		PublicIPv4:  ip,
		Status:      string(srv.Status),
		ServerType:  srv.ServerType.Name,
		CPUCores:    srv.ServerType.Cores,
		RAMMB:       int(srv.ServerType.Memory * 1024),
		MonthlyCost: parseMonthlyCost(srv.ServerType.Pricings),
	}, nil
}

// parseMonthlyCost extracts monthly gross cost from server type pricing.
func parseMonthlyCost(pricings []hcloud.ServerTypeLocationPricing) float64 {
	if len(pricings) == 0 {
		return 0.0
	}
	f, _ := strconv.ParseFloat(pricings[0].Monthly.Gross, 64)
	return f
}
