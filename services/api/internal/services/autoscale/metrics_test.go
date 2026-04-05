package autoscale_test

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/services/autoscale"
)

func TestNewK8sMetricsProvider(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	provider := autoscale.NewK8sMetricsProvider(client)
	if provider == nil {
		t.Fatal("Expected non-nil provider")
	}
}

func TestGetClusterMetrics_MemoryClient(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	provider := autoscale.NewK8sMetricsProvider(client)
	ctx := context.Background()

	cpu, ram, err := provider.GetClusterMetrics(ctx)
	if err != nil {
		t.Fatalf("GetClusterMetrics failed: %v", err)
	}
	// Memory client returns 50% utilization (2000/4000 CPU, 4GB/8GB RAM)
	if cpu != 50.0 {
		t.Errorf("Expected CPU 50.0%%, got %f%%", cpu)
	}
	if ram != 50.0 {
		t.Errorf("Expected RAM 50.0%%, got %f%%", ram)
	}
}
