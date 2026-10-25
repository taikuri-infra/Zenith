package k8sclient

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodLogOptions selects which log lines ReadPodLogs returns.
type PodLogOptions struct {
	Since     time.Duration // 0 = from the start of the retained log
	TailLines int64         // 0 = no limit
	Follow    bool
}

// PodLogReader is implemented by clients that can read container logs. It is
// separate from Client so fakes of Client do not have to grow a log method.
type PodLogReader interface {
	// ReadPodLogs returns timestamped log lines of the oldest running pod that
	// matches podSelector. ErrNoPods means nothing matched.
	ReadPodLogs(ctx context.Context, namespace, podSelector string, opts PodLogOptions) (io.ReadCloser, error)
}

// ErrNoPods is returned when no pod matches the selector.
var ErrNoPods = fmt.Errorf("no pods match the selector")

func (c *RealClient) ReadPodLogs(ctx context.Context, namespace, podSelector string, opts PodLogOptions) (io.ReadCloser, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: podSelector})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	var pick *corev1.Pod
	for i := range pods.Items {
		p := &pods.Items[i]
		if p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodSucceeded && p.Status.Phase != corev1.PodFailed {
			continue
		}
		if pick == nil || p.CreationTimestamp.Before(&pick.CreationTimestamp) {
			pick = p
		}
	}
	if pick == nil {
		return nil, ErrNoPods
	}
	lo := &corev1.PodLogOptions{Timestamps: true, Follow: opts.Follow}
	if opts.TailLines > 0 {
		lo.TailLines = &opts.TailLines
	}
	if opts.Since > 0 {
		secs := int64(opts.Since.Seconds())
		if secs < 1 {
			secs = 1
		}
		lo.SinceSeconds = &secs
	}
	return c.clientset.CoreV1().Pods(namespace).GetLogs(pick.Name, lo).Stream(ctx)
}

// SetPodLogs lets tests of MemoryClient serve canned logs per selector.
func (c *MemoryClient) SetPodLogs(selector, logs string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.podLogs == nil {
		c.podLogs = map[string]string{}
	}
	c.podLogs[selector] = logs
}

func (c *MemoryClient) ReadPodLogs(_ context.Context, _, podSelector string, _ PodLogOptions) (io.ReadCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	logs, ok := c.podLogs[podSelector]
	if !ok {
		return nil, ErrNoPods
	}
	return io.NopCloser(strings.NewReader(logs)), nil
}
