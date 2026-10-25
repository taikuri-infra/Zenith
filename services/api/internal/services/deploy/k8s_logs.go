package deploy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// K8sLogSource reads app logs straight from the app's pod (stdout+stderr), so
// logs work without a Loki install.
type K8sLogSource struct {
	client    k8sclient.PodLogReader
	namespace string
}

func NewK8sLogSource(c k8sclient.PodLogReader, namespace string) *K8sLogSource {
	return &K8sLogSource{client: c, namespace: namespace}
}

func appPodSelector(app *entities.App) string { return "zenith.dev/app-id=" + app.ID }

// parseLogLine splits the RFC3339Nano timestamp Kubernetes prepends.
func parseLogLine(line string) dto.MonitoringLogEntry {
	ts, rest, ok := strings.Cut(line, " ")
	if t, err := time.Parse(time.RFC3339Nano, ts); ok && err == nil {
		return dto.MonitoringLogEntry{Timestamp: t, Line: rest}
	}
	return dto.MonitoringLogEntry{Timestamp: time.Now(), Line: line}
}

func (s *K8sLogSource) Tail(ctx context.Context, app *entities.App, since time.Duration, limit int) ([]dto.MonitoringLogEntry, error) {
	r, err := s.client.ReadPodLogs(ctx, s.namespace, appPodSelector(app), k8sclient.PodLogOptions{Since: since, TailLines: int64(limit)})
	if err != nil {
		if errors.Is(err, k8sclient.ErrNoPods) {
			return []dto.MonitoringLogEntry{}, nil
		}
		return nil, fmt.Errorf("pod logs %s: %w", app.Name, err)
	}
	defer r.Close()
	out := []dto.MonitoringLogEntry{}
	for _, line := range readLines(r) {
		out = append(out, parseLogLine(line))
	}
	return out, nil
}

func (s *K8sLogSource) Stream(ctx context.Context, app *entities.App, out chan<- dto.MonitoringLogEntry) error {
	r, err := s.client.ReadPodLogs(ctx, s.namespace, appPodSelector(app), k8sclient.PodLogOptions{Follow: true, TailLines: 50})
	if err != nil {
		return err
	}
	defer r.Close()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		select {
		case out <- parseLogLine(sc.Text()):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func readLines(r io.Reader) []string {
	var lines []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
