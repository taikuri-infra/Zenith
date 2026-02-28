package k8sclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RealClient implements k8s.Client using the official Kubernetes client-go library.
// It auto-detects in-cluster vs local kubeconfig.
type RealClient struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
}

// NewRealClient creates a RealClient, auto-detecting in-cluster or local kubeconfig.
func NewRealClient() (*RealClient, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to local kubeconfig
		home, _ := os.UserHomeDir()
		kubeconfigPath := filepath.Join(home, ".kube", "config")

		// Allow override via KUBECONFIG env var
		if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
			kubeconfigPath = envPath
		}

		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build k8s config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &RealClient{
		clientset:     clientset,
		dynamicClient: dynClient,
	}, nil
}

// gvrFromCRD maps a CRD Kind to a GroupVersionResource.
// Zenith CRDs use apiVersion "zenith.dev/v1alpha1".
func gvrFromCRD(obj *CRDObject) schema.GroupVersionResource {
	parts := strings.SplitN(obj.APIVersion, "/", 2)
	group := ""
	version := "v1"
	if len(parts) == 2 {
		group = parts[0]
		version = parts[1]
	}

	// Pluralize kind (simple lowercase + "s")
	plural := strings.ToLower(obj.Kind) + "s"

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: plural,
	}
}

func gvrFromKind(kind, namespace string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "zenith.dev",
		Version:  "v1alpha1",
		Resource: strings.ToLower(kind) + "s",
	}
}

// --- CRD methods ---

func (c *RealClient) CreateCRD(ctx context.Context, obj *CRDObject) error {
	gvr := gvrFromCRD(obj)
	uObj, err := crdToUnstructured(obj)
	if err != nil {
		return err
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(obj.Metadata.Namespace).Create(ctx, uObj, metav1.CreateOptions{})
	return err
}

func (c *RealClient) GetCRD(ctx context.Context, kind, namespace, name string) (*CRDObject, error) {
	gvr := gvrFromKind(kind, namespace)
	uObj, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return unstructuredToCRD(uObj)
}

func (c *RealClient) UpdateCRD(ctx context.Context, obj *CRDObject) error {
	gvr := gvrFromCRD(obj)
	uObj, err := crdToUnstructured(obj)
	if err != nil {
		return err
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(obj.Metadata.Namespace).Update(ctx, uObj, metav1.UpdateOptions{})
	return err
}

func (c *RealClient) DeleteCRD(ctx context.Context, kind, namespace, name string) error {
	gvr := gvrFromKind(kind, namespace)
	return c.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *RealClient) ListCRDs(ctx context.Context, kind, namespace string) ([]*CRDObject, error) {
	gvr := gvrFromKind(kind, namespace)
	list, err := c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []*CRDObject
	for i := range list.Items {
		obj, err := unstructuredToCRD(&list.Items[i])
		if err != nil {
			continue // skip invalid items
		}
		result = append(result, obj)
	}

	return result, nil
}

// --- Job methods ---

// CreateJob submits a Kubernetes Job using the dynamic client.
// job.Spec contains the full batch/v1 Job manifest (from ToK8sJobManifest()),
// which is applied as an unstructured object to avoid deserialization issues.
func (c *RealClient) CreateJob(ctx context.Context, job *JobObject) error {
	// Convert the full Job manifest map to an unstructured object
	manifest := job.Spec
	if manifest == nil {
		return fmt.Errorf("job spec is nil")
	}

	uObj := &unstructured.Unstructured{Object: manifest}

	gvr := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}

	_, err := c.dynamicClient.Resource(gvr).Namespace(job.Namespace).Create(ctx, uObj, metav1.CreateOptions{})
	return err
}

func (c *RealClient) GetJob(ctx context.Context, namespace, name string) (*JobObject, error) {
	k8sJob, err := c.clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &JobObject{
		Name:      k8sJob.Name,
		Namespace: k8sJob.Namespace,
		Labels:    k8sJob.Labels,
		Active:    k8sJob.Status.Active,
		Succeeded: k8sJob.Status.Succeeded,
		Failed:    k8sJob.Status.Failed,
	}, nil
}

func (c *RealClient) DeleteJob(ctx context.Context, namespace, name string) error {
	propagation := metav1.DeletePropagationForeground
	return c.clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

// GetPodLogs streams log lines from the first pod matching podSelector into logCh.
func (c *RealClient) GetPodLogs(ctx context.Context, namespace, podSelector string, logCh chan<- string) error {
	defer close(logCh)

	// Find pods matching the label selector
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: podSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found matching selector %s", podSelector)
	}

	podName := pods.Items[0].Name

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open log stream: %w", err)
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case logCh <- scanner.Text():
		}
	}

	return scanner.Err()
}

// --- ConfigMap methods ---

func (c *RealClient) CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
	_, err := c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func (c *RealClient) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	return c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// --- Conversion helpers ---

func crdToUnstructured(obj *CRDObject) (*unstructured.Unstructured, error) {
	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: data}, nil
}

func unstructuredToCRD(uObj *unstructured.Unstructured) (*CRDObject, error) {
	raw, err := json.Marshal(uObj.Object)
	if err != nil {
		return nil, err
	}

	var obj CRDObject
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}
