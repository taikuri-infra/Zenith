package hetzner_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/hetzner"
)

// mockHetznerAPI is a test double for HetznerAPI.
type mockHetznerAPI struct {
	servers   map[int64]*hetzner.ServerResult
	nextID    int64
	createErr error
	deleteErr error
	listErr   error
	getErr    error
}

func newMockHetznerAPI() *mockHetznerAPI {
	return &mockHetznerAPI{
		servers: make(map[int64]*hetzner.ServerResult),
		nextID:  1000,
	}
}

func (m *mockHetznerAPI) CreateServer(_ context.Context, name, serverType, location, _ string) (*hetzner.ServerResult, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.nextID++
	srv := &hetzner.ServerResult{
		ID:          m.nextID,
		Name:        name,
		PublicIPv4:  fmt.Sprintf("10.0.0.%d", m.nextID%256),
		Status:      "running",
		ServerType:  serverType,
		CPUCores:    4,
		RAMMB:       8192,
		MonthlyCost: 15.59,
	}
	m.servers[srv.ID] = srv
	return srv, nil
}

func (m *mockHetznerAPI) DeleteServer(_ context.Context, serverID int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.servers[serverID]; !ok {
		return fmt.Errorf("server %d not found", serverID)
	}
	delete(m.servers, serverID)
	return nil
}

func (m *mockHetznerAPI) ListServers(_ context.Context) ([]hetzner.ServerResult, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	result := make([]hetzner.ServerResult, 0, len(m.servers))
	for _, s := range m.servers {
		result = append(result, *s)
	}
	return result, nil
}

func (m *mockHetznerAPI) GetServer(_ context.Context, serverID int64) (*hetzner.ServerResult, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	s, ok := m.servers[serverID]
	if !ok {
		return nil, fmt.Errorf("server %d not found", serverID)
	}
	cpy := *s
	return &cpy, nil
}

func TestCreateServer(t *testing.T) {
	mock := newMockHetznerAPI()
	ctx := context.Background()

	srv, err := mock.CreateServer(ctx, "zenith-worker-1", "cpx31", "fsn1", "#!/bin/bash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv.Name != "zenith-worker-1" {
		t.Errorf("expected name zenith-worker-1, got %s", srv.Name)
	}
	if srv.Status != "running" {
		t.Errorf("expected status running, got %s", srv.Status)
	}
	if len(mock.servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(mock.servers))
	}
}

func TestCreateServerError(t *testing.T) {
	mock := newMockHetznerAPI()
	mock.createErr = fmt.Errorf("API limit reached")
	ctx := context.Background()

	_, err := mock.CreateServer(ctx, "test", "cpx31", "fsn1", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteServer(t *testing.T) {
	mock := newMockHetznerAPI()
	ctx := context.Background()

	srv, _ := mock.CreateServer(ctx, "zenith-worker-1", "cpx31", "fsn1", "")
	if err := mock.DeleteServer(ctx, srv.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(mock.servers))
	}
}

func TestDeleteServerNotFound(t *testing.T) {
	mock := newMockHetznerAPI()
	ctx := context.Background()

	if err := mock.DeleteServer(ctx, 9999); err == nil {
		t.Fatal("expected error for missing server")
	}
}

func TestListServers(t *testing.T) {
	mock := newMockHetznerAPI()
	ctx := context.Background()

	mock.CreateServer(ctx, "w-1", "cpx31", "fsn1", "")
	mock.CreateServer(ctx, "w-2", "cpx31", "fsn1", "")

	servers, err := mock.ListServers(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}
}

func TestGetServer(t *testing.T) {
	mock := newMockHetznerAPI()
	ctx := context.Background()

	created, _ := mock.CreateServer(ctx, "w-1", "cpx31", "fsn1", "")
	got, err := mock.GetServer(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "w-1" {
		t.Errorf("expected name w-1, got %s", got.Name)
	}
}

func TestGetServerNotFound(t *testing.T) {
	mock := newMockHetznerAPI()
	ctx := context.Background()

	_, err := mock.GetServer(ctx, 9999)
	if err == nil {
		t.Fatal("expected error for missing server")
	}
}

// Ensure mockHetznerAPI satisfies HetznerAPI
var _ hetzner.HetznerAPI = (*mockHetznerAPI)(nil)
