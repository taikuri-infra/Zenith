package autoscale

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/hetznerclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// MetricsProvider returns aggregate cluster CPU/RAM utilization.
type MetricsProvider interface {
	GetClusterMetrics(ctx context.Context) (cpuPercent, ramPercent float64, err error)
}

// Autoscaler monitors cluster utilization and scales Hetzner worker nodes.
type Autoscaler struct {
	hetznerClient hetznerclient.HetznerAPI
	metrics       MetricsProvider
	repo          ports.AutoscaleRepository
	adminRepo     ports.AdminRepository
	config        entities.AutoscalerConfig
	k3sJoinToken  string
	k3sServerURL  string

	lastScaleUp   time.Time
	lastScaleDown time.Time
	stopCh        chan struct{}
}

// NewAutoscaler creates a new Autoscaler.
func NewAutoscaler(
	hetznerClient hetznerclient.HetznerAPI,
	metrics MetricsProvider,
	repo ports.AutoscaleRepository,
	adminRepo ports.AdminRepository,
	config entities.AutoscalerConfig,
	k3sJoinToken string,
	k3sServerURL string,
) *Autoscaler {
	return &Autoscaler{
		hetznerClient: hetznerClient,
		metrics:       metrics,
		repo:          repo,
		adminRepo:     adminRepo,
		config:        config,
		k3sJoinToken:  k3sJoinToken,
		k3sServerURL:  k3sServerURL,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the autoscaler background loop.
func (a *Autoscaler) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-a.stopCh:
				return
			case <-ticker.C:
				a.CheckOnce()
			}
		}
	}()
}

// Stop gracefully shuts down the autoscaler.
func (a *Autoscaler) Stop() {
	close(a.stopCh)
}

// CheckOnce runs a single autoscaler evaluation cycle.
func (a *Autoscaler) CheckOnce() {
	ctx := context.Background()

	// 1. List current managed nodes
	servers, err := a.hetznerClient.ListServers(ctx)
	if err != nil {
		slog.Error("autoscaler failed to list servers", "error", err)
		return
	}
	nodeCount := len(servers)

	// 2. Get cluster metrics
	cpuPct, ramPct, err := a.metrics.GetClusterMetrics(ctx)
	if err != nil {
		slog.Error("autoscaler failed to get metrics", "error", err)
		return
	}

	// 3. Update status in repo
	budgetUsed := 0.0
	for _, s := range servers {
		budgetUsed += s.MonthlyCost
	}
	_ = a.repo.UpdateStatus(ctx, &entities.AutoscalerStatus{
		Enabled:       true,
		NodeCount:     nodeCount,
		MinNodes:      a.config.MinNodes,
		MaxNodes:      a.config.MaxNodes,
		CPUPercent:    cpuPct,
		RAMPercent:    ramPct,
		BudgetCapEUR:  a.config.BudgetCapEUR,
		BudgetUsedEUR: budgetUsed,
		LastScaleUp:   a.lastScaleUp,
		LastScaleDown: a.lastScaleDown,
		LastCheckAt:   time.Now(),
	})

	// 4. Check if scale-up is needed
	if (cpuPct > a.config.ScaleUpCPU || ramPct > a.config.ScaleUpRAM) &&
		nodeCount < a.config.MaxNodes &&
		time.Since(a.lastScaleUp) >= a.config.CooldownUp &&
		budgetUsed < a.config.BudgetCapEUR {
		a.scaleUp(ctx, nodeCount, cpuPct, ramPct)
		return
	}

	// 5. Check if scale-down is needed
	if cpuPct < a.config.ScaleDownCPU && ramPct < a.config.ScaleDownRAM &&
		nodeCount > a.config.MinNodes &&
		time.Since(a.lastScaleDown) >= a.config.CooldownDown {
		a.scaleDown(ctx, servers, nodeCount, cpuPct, ramPct)
		return
	}
}

// scaleUp creates a new Hetzner worker node.
func (a *Autoscaler) scaleUp(ctx context.Context, nodeCount int, cpuPct, ramPct float64) {
	name := fmt.Sprintf("zenith-worker-%d", time.Now().UnixMilli())

	userData := fmt.Sprintf(`#!/bin/bash
set -e
curl -sfL https://get.k3s.io | K3S_URL="%s" K3S_TOKEN="%s" sh -s - agent
`, a.k3sServerURL, a.k3sJoinToken)

	srv, err := a.hetznerClient.CreateServer(ctx, name, a.config.ServerType, a.config.Location, userData)
	if err != nil {
		slog.Error("autoscaler scale-up failed", "error", err)
		return
	}

	a.lastScaleUp = time.Now()

	// Save node record
	_ = a.repo.SaveNode(ctx, &entities.HetznerNode{
		ServerID:    srv.ID,
		Name:        srv.Name,
		IP:          srv.PublicIPv4,
		Status:      "running",
		ServerType:  srv.ServerType,
		CPUCores:    srv.CPUCores,
		RAMMB:       srv.RAMMB,
		MonthlyCost: srv.MonthlyCost,
		CreatedAt:   time.Now(),
	})

	reason := fmt.Sprintf("CPU=%.0f%% RAM=%.0f%% (thresholds: CPU>%.0f%% or RAM>%.0f%%)",
		cpuPct, ramPct, a.config.ScaleUpCPU, a.config.ScaleUpRAM)

	_ = a.repo.LogScaleEvent(ctx, &entities.AutoscaleEvent{
		Timestamp:  time.Now(),
		Action:     entities.AutoscaleActionScaleUp,
		OldCount:   nodeCount,
		NewCount:   nodeCount + 1,
		Reason:     reason,
		ServerName: name,
	})

	_ = a.adminRepo.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  "autoscaler",
		Action: fmt.Sprintf("Scaled up: created %s (%s)", name, reason),
	})

	slog.Info("autoscaler scale-up: created server", "name", name, "id", srv.ID, "ip", srv.PublicIPv4)
}

// scaleDown removes the least-utilized node.
func (a *Autoscaler) scaleDown(ctx context.Context, servers []hetznerclient.ServerResult, nodeCount int, cpuPct, ramPct float64) {
	if len(servers) == 0 {
		return
	}

	// Pick the last server (newest) to remove for simplicity
	target := servers[len(servers)-1]

	if err := a.hetznerClient.DeleteServer(ctx, target.ID); err != nil {
		slog.Error("autoscaler scale-down failed", "error", err)
		return
	}

	a.lastScaleDown = time.Now()

	_ = a.repo.DeleteNode(ctx, target.ID)

	reason := fmt.Sprintf("CPU=%.0f%% RAM=%.0f%% (thresholds: CPU<%.0f%% and RAM<%.0f%%)",
		cpuPct, ramPct, a.config.ScaleDownCPU, a.config.ScaleDownRAM)

	_ = a.repo.LogScaleEvent(ctx, &entities.AutoscaleEvent{
		Timestamp:  time.Now(),
		Action:     entities.AutoscaleActionScaleDown,
		OldCount:   nodeCount,
		NewCount:   nodeCount - 1,
		Reason:     reason,
		ServerName: target.Name,
	})

	_ = a.adminRepo.AddAuditEntry(ctx, entities.AuditEntry{
		Time:   time.Now().Format("15:04"),
		Actor:  "autoscaler",
		Action: fmt.Sprintf("Scaled down: removed %s (%s)", target.Name, reason),
	})

	slog.Info("autoscaler scale-down: removed server", "name", target.Name, "id", target.ID)
}
