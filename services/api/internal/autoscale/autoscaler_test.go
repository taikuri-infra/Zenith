package autoscale_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/autoscale"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/hetzner"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"time"
)

// --- mock HetznerAPI ---

type mockHetzner struct {
	servers   map[int64]*hetzner.ServerResult
	nextID    int64
	createErr error
}

func newMockHetzner() *mockHetzner {
	return &mockHetzner{servers: make(map[int64]*hetzner.ServerResult), nextID: 1000}
}

func (m *mockHetzner) CreateServer(_ context.Context, name, serverType, _, _ string) (*hetzner.ServerResult, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.nextID++
	s := &hetzner.ServerResult{
		ID: m.nextID, Name: name, PublicIPv4: "10.0.0.1",
		Status: "running", ServerType: serverType,
		CPUCores: 4, RAMMB: 8192, MonthlyCost: 15.59,
	}
	m.servers[s.ID] = s
	return s, nil
}

func (m *mockHetzner) DeleteServer(_ context.Context, id int64) error {
	delete(m.servers, id)
	return nil
}

func (m *mockHetzner) ListServers(_ context.Context) ([]hetzner.ServerResult, error) {
	out := make([]hetzner.ServerResult, 0, len(m.servers))
	for _, s := range m.servers {
		out = append(out, *s)
	}
	return out, nil
}

func (m *mockHetzner) GetServer(_ context.Context, id int64) (*hetzner.ServerResult, error) {
	s, ok := m.servers[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	cpy := *s
	return &cpy, nil
}

// --- mock MetricsProvider ---

type mockMetrics struct {
	cpu float64
	ram float64
	err error
}

func (m *mockMetrics) GetClusterMetrics(_ context.Context) (float64, float64, error) {
	return m.cpu, m.ram, m.err
}

// --- mock AdminRepository (minimal) ---

type mockAdminRepo struct{}

func (r *mockAdminRepo) GetSettings(_ context.Context) (*entities.PlatformSettings, error)                     { return &entities.PlatformSettings{}, nil }
func (r *mockAdminRepo) UpdateSettings(_ context.Context, _ *entities.PlatformSettings) (*entities.PlatformSettings, error) { return &entities.PlatformSettings{}, nil }
func (r *mockAdminRepo) ListModules(_ context.Context) ([]entities.Module, error)                              { return nil, nil }
func (r *mockAdminRepo) GetModule(_ context.Context, _ string) (*entities.Module, error)                       { return nil, nil }
func (r *mockAdminRepo) UpdateModule(_ context.Context, _ string) (*entities.Module, error)                    { return nil, nil }
func (r *mockAdminRepo) ListAuditLog(_ context.Context, _, _ int) ([]entities.AuditEntry, error)               { return nil, nil }
func (r *mockAdminRepo) AddAuditEntry(_ context.Context, _ entities.AuditEntry) error                         { return nil }
func (r *mockAdminRepo) GetPlatformUpdate(_ context.Context) (*entities.PlatformUpdate, error)                 { return nil, nil }
func (r *mockAdminRepo) ListUpdateHistory(_ context.Context) ([]entities.UpdateHistoryEntry, error)            { return nil, nil }

// --- helpers ---

func defaultConfig() entities.AutoscalerConfig {
	return entities.AutoscalerConfig{
		MinNodes:     2,
		MaxNodes:     10,
		ScaleUpCPU:   80,
		ScaleUpRAM:   80,
		ScaleDownCPU: 40,
		ScaleDownRAM: 40,
		CooldownUp:   0,
		CooldownDown: 0,
		BudgetCapEUR: 450,
		ServerType:   "cpx31",
		Location:     "fsn1",
	}
}

func buildAutoscaler(h *mockHetzner, m *mockMetrics, cfg entities.AutoscalerConfig) (*autoscale.Autoscaler, store.AutoscaleRepository) {
	repo := store.NewMemoryAutoscaleRepository()
	admin := &mockAdminRepo{}
	as := autoscale.NewAutoscaler(h, m, repo, admin, cfg, "test-token", "https://k3s.example.com:6443")
	return as, repo
}

// --- tests ---

func TestScaleUpTriggered(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 3; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 85, ram: 75} // CPU > 80%
	as, repo := buildAutoscaler(h, m, defaultConfig())

	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 4 {
		t.Errorf("expected 4 servers after scale-up, got %d", len(servers))
	}

	events, _ := repo.ListScaleEvents(context.Background(), 10)
	if len(events) == 0 {
		t.Fatal("expected at least one scale event")
	}
	if events[0].Action != entities.AutoscaleActionScaleUp {
		t.Errorf("expected scale_up event, got %s", events[0].Action)
	}
}

func TestScaleDownTriggered(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 4; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 30, ram: 25} // Both below 40%
	as, repo := buildAutoscaler(h, m, defaultConfig())

	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 3 {
		t.Errorf("expected 3 servers after scale-down, got %d", len(servers))
	}

	events, _ := repo.ListScaleEvents(context.Background(), 10)
	if len(events) == 0 {
		t.Fatal("expected at least one scale event")
	}
	if events[0].Action != entities.AutoscaleActionScaleDown {
		t.Errorf("expected scale_down event, got %s", events[0].Action)
	}
}

func TestCooldownPreventsFlapping(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 3; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 85, ram: 85}
	cfg := defaultConfig()
	cfg.CooldownUp = 1 * time.Hour

	as, _ := buildAutoscaler(h, m, cfg)

	// First check scales up
	as.CheckOnce()
	// Second check should be blocked by cooldown
	as.CheckOnce()
	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 4 {
		t.Errorf("expected exactly 4 servers (cooldown prevents more), got %d", len(servers))
	}
}

func TestMinNodesRespected(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 2; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 10, ram: 10}
	cfg := defaultConfig()
	cfg.MinNodes = 2

	as, _ := buildAutoscaler(h, m, cfg)
	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 2 {
		t.Errorf("expected 2 servers (min), got %d", len(servers))
	}
}

func TestMaxNodesRespected(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 10; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 95, ram: 95}
	cfg := defaultConfig()
	cfg.MaxNodes = 10

	as, _ := buildAutoscaler(h, m, cfg)
	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 10 {
		t.Errorf("expected 10 servers (max), got %d", len(servers))
	}
}

func TestBudgetCapRespected(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 3; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 90, ram: 90}
	cfg := defaultConfig()
	cfg.BudgetCapEUR = 46 // Below cost of 3 servers × 15.59

	as, _ := buildAutoscaler(h, m, cfg)
	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 3 {
		t.Errorf("expected 3 servers (budget cap), got %d", len(servers))
	}
}

func TestNoActionInNormalRange(t *testing.T) {
	h := newMockHetzner()
	for i := 0; i < 4; i++ {
		h.CreateServer(context.Background(), fmt.Sprintf("w-%d", i), "cpx31", "fsn1", "")
	}

	m := &mockMetrics{cpu: 60, ram: 55} // Between thresholds
	as, repo := buildAutoscaler(h, m, defaultConfig())
	as.CheckOnce()

	servers, _ := h.ListServers(context.Background())
	if len(servers) != 4 {
		t.Errorf("expected 4 servers (no change), got %d", len(servers))
	}

	events, _ := repo.ListScaleEvents(context.Background(), 10)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestStartStop(t *testing.T) {
	h := newMockHetzner()
	m := &mockMetrics{cpu: 50, ram: 50}
	as, _ := buildAutoscaler(h, m, defaultConfig())

	as.Start(100 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	as.Stop()
	// Should not panic
}
