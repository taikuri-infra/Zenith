package k8sclient

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// Type aliases: canonical definitions live in ports/infrastructure.go.
// These aliases ensure backward compatibility — all existing code continues
// to use k8sclient.CRDObject etc. without changes.
type CRDObject = ports.K8sCRDObject
type ObjectMeta = ports.K8sObjectMeta
type JobObject = ports.K8sJobObject
type LimitRangeSpec = ports.K8sLimitRangeSpec

// Client provides an interface for Kubernetes operations.
// In production, this wraps a real K8s client. For testing, use MemoryClient.
type Client interface {
	// CRD operations (Deployment, Service, IngressRoute, etc.)
	CreateCRD(ctx context.Context, obj *CRDObject) error
	GetCRD(ctx context.Context, kind, namespace, name string) (*CRDObject, error)
	UpdateCRD(ctx context.Context, obj *CRDObject) error
	PatchCRD(ctx context.Context, obj *CRDObject) error
	DeleteCRD(ctx context.Context, kind, namespace, name string) error
	ListCRDs(ctx context.Context, kind, namespace string) ([]*CRDObject, error)

	// Job operations (Kaniko build jobs)
	CreateJob(ctx context.Context, job *JobObject) error
	GetJob(ctx context.Context, namespace, name string) (*JobObject, error)
	DeleteJob(ctx context.Context, namespace, name string) error
	// GetPodLogs streams log lines from the first pod matching podSelector into logCh.
	// The channel is closed when streaming is complete or the context is cancelled.
	GetPodLogs(ctx context.Context, namespace, podSelector string, logCh chan<- string) error

	// ConfigMap operations (generated Dockerfiles for Kaniko builds)
	CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) error
	DeleteConfigMap(ctx context.Context, namespace, name string) error

	// Namespace operations (tenant provisioning)
	CreateNamespace(ctx context.Context, name string, labels map[string]string) error
	GetNamespace(ctx context.Context, name string) error
	DeleteNamespace(ctx context.Context, name string) error

	// Secret operations (tenant credentials)
	CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error
	GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error)
	DeleteSecret(ctx context.Context, namespace, name string) error

	// ResourceQuota operations (tenant limits)
	CreateResourceQuota(ctx context.Context, namespace, name string, hard map[string]string) error

	// LimitRange operations (container defaults)
	CreateLimitRange(ctx context.Context, namespace, name string, limits LimitRangeSpec) error

	// Generic CRD operations with explicit apiVersion (for non-Zenith CRDs)
	GetCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) (*CRDObject, error)
	DeleteCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) error
}

// MemoryClient is an in-memory K8s client for testing and development.
type MemoryClient struct {
	mu         sync.RWMutex
	objects    map[string]*CRDObject
	jobs       map[string]*JobObject
	namespaces map[string]map[string]string
	secrets    map[string]map[string][]byte
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{
		objects:    make(map[string]*CRDObject),
		jobs:       make(map[string]*JobObject),
		namespaces: make(map[string]map[string]string),
		secrets:    make(map[string]map[string][]byte),
	}
}

func objectKey(kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

func jobKey(namespace, name string) string {
	return namespace + "/" + name
}

// --- CRD methods ---

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

func (c *MemoryClient) PatchCRD(ctx context.Context, obj *CRDObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := objectKey(obj.Kind, obj.Metadata.Namespace, obj.Metadata.Name)
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

// --- Job methods ---

// CreateJob stores a job and immediately marks it as Succeeded (fake execution).
func (c *MemoryClient) CreateJob(ctx context.Context, job *JobObject) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := jobKey(job.Namespace, job.Name)
	if _, exists := c.jobs[key]; exists {
		return fmt.Errorf("job %s already exists", key)
	}

	// Simulate immediate success in memory mode
	job.Succeeded = 1
	c.jobs[key] = job
	return nil
}

// GetJob returns a stored job.
func (c *MemoryClient) GetJob(ctx context.Context, namespace, name string) (*JobObject, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := jobKey(namespace, name)
	job, exists := c.jobs[key]
	if !exists {
		return nil, fmt.Errorf("job %s not found", key)
	}

	return job, nil
}

// DeleteJob removes a job from memory.
func (c *MemoryClient) DeleteJob(ctx context.Context, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := jobKey(namespace, name)
	if _, exists := c.jobs[key]; !exists {
		return fmt.Errorf("job %s not found", key)
	}

	delete(c.jobs, key)
	return nil
}

// CreateConfigMap is a no-op in memory mode.
func (c *MemoryClient) CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) error {
	return nil
}

// DeleteConfigMap is a no-op in memory mode.
func (c *MemoryClient) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	return nil
}

// --- Namespace methods ---

func (c *MemoryClient) CreateNamespace(ctx context.Context, name string, labels map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.namespaces[name]; exists {
		return fmt.Errorf("namespace %s already exists", name)
	}
	c.namespaces[name] = labels
	return nil
}

func (c *MemoryClient) GetNamespace(ctx context.Context, name string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, exists := c.namespaces[name]; !exists {
		return fmt.Errorf("namespace %s not found", name)
	}
	return nil
}

func (c *MemoryClient) DeleteNamespace(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.namespaces, name)
	return nil
}

// --- Secret methods ---

func (c *MemoryClient) CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := namespace + "/" + name
	c.secrets[key] = data
	return nil
}

func (c *MemoryClient) GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := namespace + "/" + name
	data, exists := c.secrets[key]
	if !exists {
		return nil, fmt.Errorf("secret %s not found", key)
	}
	return data, nil
}

func (c *MemoryClient) DeleteSecret(ctx context.Context, namespace, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.secrets, namespace+"/"+name)
	return nil
}

// --- ResourceQuota / LimitRange (no-op in memory) ---

func (c *MemoryClient) CreateResourceQuota(ctx context.Context, namespace, name string, hard map[string]string) error {
	return nil
}

func (c *MemoryClient) CreateLimitRange(ctx context.Context, namespace, name string, limits LimitRangeSpec) error {
	return nil
}

// --- Generic CRD with explicit apiVersion ---

func (c *MemoryClient) GetCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) (*CRDObject, error) {
	return c.GetCRD(ctx, kind, namespace, name)
}

func (c *MemoryClient) DeleteCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) error {
	return c.DeleteCRD(ctx, kind, namespace, name)
}

// GetPodLogs sends fake build output lines (dev/test mode).
func (c *MemoryClient) GetPodLogs(ctx context.Context, namespace, podSelector string, logCh chan<- string) error {
	defer close(logCh)

	fakeLines := []string{
		"INFO[0001] Retrieving image manifest golang:1.25-alpine",
		"INFO[0003] Executing 0 build triggers",
		"INFO[0005] Building stage 'builder' [idx: '0', base-idx: '-1']",
		"INFO[0012] RUN go build -o /app/server .",
		"INFO[0028] Copying dir /app to /app",
		"INFO[0030] Taking snapshot of files...",
		"INFO[0031] EXPOSE 8080",
		"INFO[0032] Pushing image to " + strings.TrimPrefix(podSelector, "zenith.dev/deployment="),
		"INFO[0035] Pushed image successfully",
	}

	for _, line := range fakeLines {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case logCh <- line:
		}
	}

	return nil
}
