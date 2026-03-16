package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// K8sProvisionerAdapter adapts k8sclient.Client to the K8sProvisioner interface
// used by ManagedServiceService for provisioning managed databases and caches.
type K8sProvisionerAdapter struct {
	client k8sclient.Client
}

// NewK8sProvisionerAdapter creates a new adapter wrapping the given k8sclient.Client.
func NewK8sProvisionerAdapter(client k8sclient.Client) *K8sProvisionerAdapter {
	return &K8sProvisionerAdapter{client: client}
}

// ApplyUnstructured creates a K8s resource from an unstructured map.
// It routes to the appropriate k8sclient method based on the kind.
func (a *K8sProvisionerAdapter) ApplyUnstructured(ctx context.Context, namespace string, obj map[string]interface{}) error {
	apiVersion, _ := obj["apiVersion"].(string)
	kind, _ := obj["kind"].(string)

	// For native K8s resources, use typed methods where available
	switch {
	case kind == "Secret":
		return a.applySecret(ctx, namespace, obj)
	case kind == "ConfigMap":
		return a.applyConfigMap(ctx, namespace, obj)
	default:
		// For CRDs and other resources (StatefulSet, Service, etc.), use dynamic client via CreateCRD
		return a.applyCRD(ctx, apiVersion, kind, namespace, obj)
	}
}

// DeleteResource deletes a K8s resource by apiVersion, kind, namespace and name.
func (a *K8sProvisionerAdapter) DeleteResource(ctx context.Context, namespace, apiVersion, kind, name string) error {
	switch {
	case kind == "Secret":
		return a.client.DeleteSecret(ctx, namespace, name)
	case kind == "ConfigMap":
		return a.client.DeleteConfigMap(ctx, namespace, name)
	default:
		return a.client.DeleteCRDWithVersion(ctx, apiVersion, kind, namespace, name)
	}
}

// GetCRDStatus retrieves the status field of a CRD object.
func (a *K8sProvisionerAdapter) GetCRDStatus(ctx context.Context, apiVersion, kind, namespace, name string) (json.RawMessage, error) {
	crd, err := a.client.GetCRDWithVersion(ctx, apiVersion, kind, namespace, name)
	if err != nil {
		return nil, err
	}
	return crd.Status, nil
}

func (a *K8sProvisionerAdapter) applySecret(ctx context.Context, namespace string, obj map[string]interface{}) error {
	metadata, _ := obj["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	data := make(map[string][]byte)
	if sd, ok := obj["stringData"].(map[string]interface{}); ok {
		for k, v := range sd {
			data[k] = []byte(fmt.Sprintf("%v", v))
		}
	}
	if bd, ok := obj["data"].(map[string]interface{}); ok {
		for k, v := range bd {
			data[k] = []byte(fmt.Sprintf("%v", v))
		}
	}

	labels := extractLabels(metadata)
	return a.client.CreateSecret(ctx, namespace, name, data, labels)
}

func (a *K8sProvisionerAdapter) applyConfigMap(ctx context.Context, namespace string, obj map[string]interface{}) error {
	metadata, _ := obj["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	data := make(map[string]string)
	if d, ok := obj["data"].(map[string]interface{}); ok {
		for k, v := range d {
			data[k] = fmt.Sprintf("%v", v)
		}
	}

	return a.client.CreateConfigMap(ctx, namespace, name, data)
}

func (a *K8sProvisionerAdapter) applyCRD(ctx context.Context, apiVersion, kind, namespace string, obj map[string]interface{}) error {
	metadata, _ := obj["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	specBytes, err := json.Marshal(obj["spec"])
	if err != nil {
		return fmt.Errorf("marshal spec: %w", err)
	}

	// Extract labels
	labels := extractLabels(metadata)

	crdObj := &ports.K8sCRDObject{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: ports.K8sObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: specBytes,
	}

	// For non-zenith CRDs, we need to use the versioned API
	if !strings.Contains(apiVersion, "zenith.dev") {
		// Use DeleteCRDWithVersion + create pattern (idempotent)
		_ = a.client.DeleteCRDWithVersion(ctx, apiVersion, kind, namespace, name)
	}

	return a.client.CreateCRD(ctx, crdObj)
}

func extractLabels(metadata map[string]interface{}) map[string]string {
	labels := make(map[string]string)
	if lbls, ok := metadata["labels"].(map[string]interface{}); ok {
		for k, v := range lbls {
			labels[k] = fmt.Sprintf("%v", v)
		}
	}
	return labels
}
