package capiclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// Compile-time check: Client implements ports.ClusterProvisioner.
var _ ports.ClusterProvisioner = (*Client)(nil)

const (
	// CAPINamespace is the management namespace where CAPI resources live.
	CAPINamespace = "zenith-system"

	// CRD kinds used for CAPI cluster resources.
	KindCluster = "Cluster"
)

// Client wraps the k8sclient.Client to provide CAPI-specific operations
// using unstructured resources (no imported CAPI Go types).
type Client struct {
	k8s k8sclient.Client
}

// NewClient creates a new CAPI client wrapping the given Kubernetes client.
func NewClient(k8sClient k8sclient.Client) *Client {
	return &Client{k8s: k8sClient}
}

// CreateCluster creates a CAPI Cluster resource in the management namespace.
func (c *Client) CreateCluster(ctx context.Context, input dto.CreateClusterInput) (*entities.Cluster, error) {
	spec, _ := json.Marshal(map[string]interface{}{
		"k8sVersion": input.K8sVersion,
		"nodes":      input.Nodes,
		"region":     input.Region,
		"type":       input.Type,
		"tenant":     input.Tenant,
	})

	crd := &k8sclient.CRDObject{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       KindCluster,
		Metadata: k8sclient.ObjectMeta{
			Name:      input.Name,
			Namespace: CAPINamespace,
			Labels: map[string]string{
				"zenith.dev/cluster-type": input.Type,
				"zenith.dev/region":       input.Region,
			},
		},
		Spec: spec,
	}

	if input.Tenant != "" {
		crd.Metadata.Labels["zenith.dev/tenant"] = input.Tenant
	}

	if err := c.k8s.CreateCRD(ctx, crd); err != nil {
		return nil, fmt.Errorf("create cluster: %w", err)
	}

	return crdToCluster(crd), nil
}

// GetCluster retrieves a single CAPI Cluster resource by name.
func (c *Client) GetCluster(ctx context.Context, name string) (*entities.Cluster, error) {
	crd, err := c.k8s.GetCRD(ctx, KindCluster, CAPINamespace, name)
	if err != nil {
		return nil, fmt.Errorf("get cluster %s: %w", name, err)
	}
	return crdToCluster(crd), nil
}

// ListClusters returns all CAPI Cluster resources in the management namespace.
func (c *Client) ListClusters(ctx context.Context) ([]entities.Cluster, error) {
	crds, err := c.k8s.ListCRDs(ctx, KindCluster, CAPINamespace)
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}

	clusters := make([]entities.Cluster, 0, len(crds))
	for _, crd := range crds {
		clusters = append(clusters, *crdToCluster(crd))
	}
	return clusters, nil
}

// DeleteCluster removes a CAPI Cluster resource by name.
func (c *Client) DeleteCluster(ctx context.Context, name string) error {
	if err := c.k8s.DeleteCRD(ctx, KindCluster, CAPINamespace, name); err != nil {
		return fmt.Errorf("delete cluster %s: %w", name, err)
	}
	return nil
}

// ScaleCluster updates the node count of a cluster.
func (c *Client) ScaleCluster(ctx context.Context, name string, nodes int) error {
	crd, err := c.k8s.GetCRD(ctx, KindCluster, CAPINamespace, name)
	if err != nil {
		return fmt.Errorf("get cluster %s for scale: %w", name, err)
	}

	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)
	spec["nodes"] = nodes
	crd.Spec, _ = json.Marshal(spec)

	if err := c.k8s.UpdateCRD(ctx, crd); err != nil {
		return fmt.Errorf("scale cluster %s: %w", name, err)
	}
	return nil
}

// UpgradeCluster updates the Kubernetes version of a cluster.
func (c *Client) UpgradeCluster(ctx context.Context, name, version string) error {
	crd, err := c.k8s.GetCRD(ctx, KindCluster, CAPINamespace, name)
	if err != nil {
		return fmt.Errorf("get cluster %s for upgrade: %w", name, err)
	}

	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)
	spec["k8sVersion"] = version
	crd.Spec, _ = json.Marshal(spec)

	// Add annotation to track upgrade
	if crd.Metadata.Annotations == nil {
		crd.Metadata.Annotations = make(map[string]string)
	}
	crd.Metadata.Annotations["zenith.dev/upgrade-target"] = version

	if err := c.k8s.UpdateCRD(ctx, crd); err != nil {
		return fmt.Errorf("upgrade cluster %s: %w", name, err)
	}
	return nil
}

// crdToCluster converts a CRDObject into a entities.Cluster.
func crdToCluster(crd *k8sclient.CRDObject) *entities.Cluster {
	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)

	k8sVersion, _ := spec["k8sVersion"].(string)
	region, _ := spec["region"].(string)
	clusterType, _ := spec["type"].(string)
	tenant, _ := spec["tenant"].(string)

	nodes := 1
	if n, ok := spec["nodes"].(float64); ok {
		nodes = int(n)
	}

	// Status fields default to placeholder values.
	// In production these would be read from the status subresource.
	cpuPercent := 0
	ramPercent := 0
	status := "healthy"

	var statusMap map[string]interface{}
	if crd.Status != nil {
		_ = json.Unmarshal(crd.Status, &statusMap)
	}
	if statusMap != nil {
		if v, ok := statusMap["cpuPercent"].(float64); ok {
			cpuPercent = int(v)
		}
		if v, ok := statusMap["ramPercent"].(float64); ok {
			ramPercent = int(v)
		}
		if v, ok := statusMap["status"].(string); ok {
			status = v
		}
	}

	upgradeAvailable, _ := crd.Metadata.Annotations["zenith.dev/upgrade-available"]

	return &entities.Cluster{
		Name:             crd.Metadata.Name,
		K8sVersion:       k8sVersion,
		Nodes:            nodes,
		Region:           region,
		Type:             clusterType,
		Tenant:           tenant,
		CPUPercent:       cpuPercent,
		RAMPercent:       ramPercent,
		Pods:             entities.ResourcePair{Used: 0, Total: 0},
		PVCs:             entities.ResourcePair{Used: 0, Total: 0},
		Status:           status,
		UpgradeAvailable: upgradeAvailable,
	}
}

