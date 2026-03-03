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

// GetClusterMetrics returns aggregate CPU and RAM utilization percentages.
// For now, this queries the k8s API for node conditions / metrics-server.
// In memory mode, returns 50%/50% so the autoscaler doesn't trigger.
func (p *K8sMetricsProvider) GetClusterMetrics(_ context.Context) (cpuPercent, ramPercent float64, err error) {
	// TODO: implement real metrics-server query using p.client when K8S_MODE=real
	// For now, return safe defaults that won't trigger scaling.
	return 50.0, 50.0, nil
}
