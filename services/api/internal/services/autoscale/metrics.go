package autoscale

import (
	"context"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
)

// K8sMetricsProvider fetches aggregate cluster metrics from Kubernetes.
// In dev/memory mode it returns safe mid-range values.
type K8sMetricsProvider struct {
	client k8sclient.Client
}

// NewK8sMetricsProvider creates a metrics provider backed by the k8s client.
func NewK8sMetricsProvider(client k8sclient.Client) *K8sMetricsProvider {
	return &K8sMetricsProvider{client: client}
}

// GetClusterMetrics returns aggregate CPU and RAM utilization percentages
// by querying node metrics from the k8s client (which in turn uses metrics-server).
func (p *K8sMetricsProvider) GetClusterMetrics(ctx context.Context) (cpuPercent, ramPercent float64, err error) {
	nodes, err := p.client.GetNodeMetrics(ctx)
	if err != nil || len(nodes) == 0 {
		// Fail safe: return mid-range values that won't trigger scaling
		return 50.0, 50.0, nil
	}

	var totalCPUCap, totalCPUUse, totalMemCap, totalMemUse int64
	for _, n := range nodes {
		totalCPUCap += n.CPUCapacityMillis
		totalCPUUse += n.CPUUsageMillis
		totalMemCap += n.MemCapacityBytes
		totalMemUse += n.MemUsageBytes
	}

	if totalCPUCap > 0 {
		cpuPercent = float64(totalCPUUse) / float64(totalCPUCap) * 100.0
	}
	if totalMemCap > 0 {
		ramPercent = float64(totalMemUse) / float64(totalMemCap) * 100.0
	}

	return cpuPercent, ramPercent, nil
}
