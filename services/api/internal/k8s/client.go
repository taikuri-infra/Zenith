package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// CRDObject represents a generic Kubernetes CRD object
type CRDObject struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   ObjectMeta        `json:"metadata"`
	Spec       json.RawMessage   `json:"spec"`
	Status     json.RawMessage   `json:"status,omitempty"`
}

type ObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Client provides an interface for Kubernetes operations.
// In production, this wraps a real K8s client. For testing, use MemoryClient.
type Client interface {
	CreateCRD(ctx context.Context, obj *CRDObject) error
	GetCRD(ctx context.Context, kind, namespace, name string) (*CRDObject, error)
	UpdateCRD(ctx context.Context, obj *CRDObject) error
	DeleteCRD(ctx context.Context, kind, namespace, name string) error
	ListCRDs(ctx context.Context, kind, namespace string) ([]*CRDObject, error)
}

// MemoryClient is an in-memory K8s client for testing and development
type MemoryClient struct {
	mu      sync.RWMutex
	objects map[string]*CRDObject
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{
		objects: make(map[string]*CRDObject),
	}
}

func objectKey(kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

func (c *MemoryClient) CreateCRD(ctx context.Context, obj *CRDObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := objectKey(obj.Kind, obj.Metadata.Namespace, obj.Metadata.Name)
	if _, exists := c.objects[key]; exists {
		return fmt.Errorf("object %s already exists", key)
	}

	c.objects[key] = obj
	return nil
}

func (c *MemoryClient) GetCRD(ctx context.Context, kind, namespace, name string) (*CRDObject, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := objectKey(kind, namespace, name)
	obj, exists := c.objects[key]
	if !exists {
		return nil, fmt.Errorf("object %s not found", key)
	}

	return obj, nil
}

func (c *MemoryClient) UpdateCRD(ctx context.Context, obj *CRDObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := objectKey(obj.Kind, obj.Metadata.Namespace, obj.Metadata.Name)
	if _, exists := c.objects[key]; !exists {
		return fmt.Errorf("object %s not found", key)
	}

	c.objects[key] = obj
	return nil
}

func (c *MemoryClient) DeleteCRD(ctx context.Context, kind, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := objectKey(kind, namespace, name)
	if _, exists := c.objects[key]; !exists {
		return fmt.Errorf("object %s not found", key)
	}

	delete(c.objects, key)
	return nil
}

func (c *MemoryClient) ListCRDs(ctx context.Context, kind, namespace string) ([]*CRDObject, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*CRDObject
	prefix := kind + "/" + namespace + "/"

	for key, obj := range c.objects {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			result = append(result, obj)
		}
	}

	return result, nil
}
