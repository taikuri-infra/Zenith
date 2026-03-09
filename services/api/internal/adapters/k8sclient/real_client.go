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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
	restConfig    *rest.Config
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
		restConfig:    cfg,
	}, nil
}

// Clientset returns the underlying kubernetes.Clientset (for pod exec).
func (c *RealClient) Clientset() *kubernetes.Clientset { return c.clientset }

// RESTConfig returns the underlying REST config (for SPDY exec).
func (c *RealClient) RESTConfig() *rest.Config { return c.restConfig }

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

func (c *RealClient) PatchCRD(ctx context.Context, obj *CRDObject) error {
	gvr := gvrFromCRD(obj)
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(obj.Metadata.Namespace).Patch(
		ctx, obj.Metadata.Name, types.MergePatchType, data, metav1.PatchOptions{},
	)
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

// --- Namespace methods ---

func (c *RealClient) CreateNamespace(ctx context.Context, name string, labels map[string]string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

func (c *RealClient) GetNamespace(ctx context.Context, name string) error {
	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	return err
}

func (c *RealClient) DeleteNamespace(ctx context.Context, name string) error {
	return c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

// --- Secret methods ---

func (c *RealClient) CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
	_, err := c.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

func (c *RealClient) GetSecret(ctx context.Context, namespace, name string) (map[string][]byte, error) {
	secret, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (c *RealClient) DeleteSecret(ctx context.Context, namespace, name string) error {
	return c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// --- ResourceQuota methods ---

func (c *RealClient) CreateResourceQuota(ctx context.Context, namespace, name string, hard map[string]string) error {
	resourceList := corev1.ResourceList{}
	for k, v := range hard {
		resourceList[corev1.ResourceName(k)] = resource.MustParse(v)
	}
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: resourceList,
		},
	}
	_, err := c.clientset.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{})
	return err
}

// --- LimitRange methods ---

func (c *RealClient) CreateLimitRange(ctx context.Context, namespace, name string, limits LimitRangeSpec) error {
	lr := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(limits.DefaultCPU),
						corev1.ResourceMemory: resource.MustParse(limits.DefaultMemory),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(limits.DefaultReqCPU),
						corev1.ResourceMemory: resource.MustParse(limits.DefaultReqMemory),
					},
					Max: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(limits.MaxCPU),
						corev1.ResourceMemory: resource.MustParse(limits.MaxMemory),
					},
					Min: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(limits.MinCPU),
						corev1.ResourceMemory: resource.MustParse(limits.MinMemory),
					},
				},
			},
		},
	}
	_, err := c.clientset.CoreV1().LimitRanges(namespace).Create(ctx, lr, metav1.CreateOptions{})
	return err
}

// --- Generic CRD with explicit apiVersion ---

// pluralizeKind converts a CRD Kind to its plural resource name.
func pluralizeKind(kind string) string {
	lower := strings.ToLower(kind)
	// Handle irregular plurals
	if strings.HasSuffix(lower, "policy") {
		return strings.TrimSuffix(lower, "y") + "ies"
	}
	if strings.HasSuffix(lower, "ingress") {
		return lower + "es"
	}
	return lower + "s"
}

func gvrFromAPIVersionKind(apiVersion, kind string) schema.GroupVersionResource {
	parts := strings.SplitN(apiVersion, "/", 2)
	group := ""
	version := "v1"
	if len(parts) == 2 {
		group = parts[0]
		version = parts[1]
	} else {
		version = parts[0]
	}
	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: pluralizeKind(kind),
	}
}

func (c *RealClient) GetCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) (*CRDObject, error) {
	gvr := gvrFromAPIVersionKind(apiVersion, kind)
	uObj, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return unstructuredToCRD(uObj)
}

func (c *RealClient) ListCRDsWithVersion(ctx context.Context, apiVersion, kind, namespace string) ([]*CRDObject, error) {
	gvr := gvrFromAPIVersionKind(apiVersion, kind)
	list, err := c.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []*CRDObject
	for i := range list.Items {
		obj, err := unstructuredToCRD(&list.Items[i])
		if err != nil {
			continue
		}
		result = append(result, obj)
	}
	return result, nil
}

func (c *RealClient) DeleteCRDWithVersion(ctx context.Context, apiVersion, kind, namespace, name string) error {
	gvr := gvrFromAPIVersionKind(apiVersion, kind)
	return c.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ListPVCs lists PersistentVolumeClaims in a namespace (or all namespaces if empty).
func (c *RealClient) ListPVCs(ctx context.Context, namespace string) ([]PVCInfo, error) {
	var pvcs []PVCInfo
	if namespace == "" {
		// List across all namespaces
		list, err := c.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pvc := range list.Items {
			pvcs = append(pvcs, pvcToInfo(pvc))
		}
	} else {
		list, err := c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pvc := range list.Items {
			pvcs = append(pvcs, pvcToInfo(pvc))
		}
	}
	return pvcs, nil
}

func pvcToInfo(pvc corev1.PersistentVolumeClaim) PVCInfo {
	size := ""
	if req, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		size = req.String()
	}
	sc := ""
	if pvc.Spec.StorageClassName != nil {
		sc = *pvc.Spec.StorageClassName
	}
	return PVCInfo{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		Size:         size,
		Status:       string(pvc.Status.Phase),
		StorageClass: sc,
	}
}

// --- Pod monitoring methods ---

// ListPods returns pods matching the label selector.
func (c *RealClient) ListPods(ctx context.Context, namespace, labelSelector string) ([]PodInfo, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	result := make([]PodInfo, 0, len(pods.Items))
	for _, p := range pods.Items {
		var restarts int32
		ready := true
		var statusReason, statusMessage string
		var lastExitCode int32
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
			if !cs.Ready {
				ready = false
			}
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				statusReason = cs.State.Waiting.Reason
				statusMessage = cs.State.Waiting.Message
			} else if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				statusReason = cs.State.Terminated.Reason
				statusMessage = cs.State.Terminated.Message
				lastExitCode = cs.State.Terminated.ExitCode
			}
			if cs.LastTerminationState.Terminated != nil {
				lastExitCode = cs.LastTerminationState.Terminated.ExitCode
				if statusMessage == "" {
					statusMessage = cs.LastTerminationState.Terminated.Message
				}
			}
		}
		startedAt := p.CreationTimestamp.Time
		if p.Status.StartTime != nil {
			startedAt = p.Status.StartTime.Time
		}
		// Sum memory limits from all containers
		var memLimit int64
		for _, cont := range p.Spec.Containers {
			if lim, ok := cont.Resources.Limits[corev1.ResourceMemory]; ok {
				memLimit += lim.Value()
			}
		}
		result = append(result, PodInfo{
			Name:             p.Name,
			Status:           string(p.Status.Phase),
			Restarts:         restarts,
			StartedAt:        startedAt,
			Ready:            ready,
			MemoryLimitBytes: memLimit,
			StatusReason:     statusReason,
			StatusMessage:    statusMessage,
			LastExitCode:     lastExitCode,
		})
	}

	return result, nil
}

// GetPodMetrics fetches resource usage from metrics-server for pods matching the label selector.
func (c *RealClient) GetPodMetrics(ctx context.Context, namespace, labelSelector string) ([]PodMetrics, error) {
	// Use the metrics.k8s.io API via raw REST request
	path := fmt.Sprintf("/apis/metrics.k8s.io/v1beta1/namespaces/%s/pods", namespace)
	if labelSelector != "" {
		path += "?labelSelector=" + labelSelector
	}

	data, err := c.clientset.RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		// Metrics-server may not be available — return empty rather than error
		return nil, nil
	}

	var metricsResp struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Containers []struct {
				Usage struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
				} `json:"usage"`
			} `json:"containers"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &metricsResp); err != nil {
		return nil, nil
	}

	result := make([]PodMetrics, 0, len(metricsResp.Items))
	for _, item := range metricsResp.Items {
		var cpuMillis int64
		var memBytes int64
		for _, cont := range item.Containers {
			cpu := resource.MustParse(cont.Usage.CPU)
			mem := resource.MustParse(cont.Usage.Memory)
			cpuMillis += cpu.MilliValue()
			memBytes += mem.Value()
		}
		result = append(result, PodMetrics{
			Name:          item.Metadata.Name,
			CPUMillicores: cpuMillis,
			MemoryBytes:   memBytes,
		})
	}

	return result, nil
}

// GetNodeMetrics fetches node-level resource usage from metrics-server and
// combines it with node capacity from the core API.
func (c *RealClient) GetNodeMetrics(ctx context.Context) ([]NodeMetrics, error) {
	// Get node capacity from core API
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil
	}

	capacityMap := make(map[string]NodeMetrics, len(nodes.Items))
	for _, node := range nodes.Items {
		cpuCap := node.Status.Capacity[corev1.ResourceCPU]
		memCap := node.Status.Capacity[corev1.ResourceMemory]
		capacityMap[node.Name] = NodeMetrics{
			Name:              node.Name,
			CPUCapacityMillis: cpuCap.MilliValue(),
			MemCapacityBytes:  memCap.Value(),
		}
	}

	// Get node usage from metrics-server
	data, err := c.clientset.RESTClient().Get().AbsPath("/apis/metrics.k8s.io/v1beta1/nodes").DoRaw(ctx)
	if err != nil {
		// Metrics-server may not be available — return capacity with zero usage
		result := make([]NodeMetrics, 0, len(capacityMap))
		for _, nm := range capacityMap {
			result = append(result, nm)
		}
		return result, nil
	}

	var metricsResp struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &metricsResp); err != nil {
		return nil, nil
	}

	// Merge usage into capacity
	for _, item := range metricsResp.Items {
		if nm, ok := capacityMap[item.Metadata.Name]; ok {
			cpu := resource.MustParse(item.Usage.CPU)
			mem := resource.MustParse(item.Usage.Memory)
			nm.CPUUsageMillis = cpu.MilliValue()
			nm.MemUsageBytes = mem.Value()
			capacityMap[item.Metadata.Name] = nm
		}
	}

	result := make([]NodeMetrics, 0, len(capacityMap))
	for _, nm := range capacityMap {
		result = append(result, nm)
	}
	return result, nil
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
