package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/lokiclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/promclient"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

const appsNamespace = "zenith-apps"

// MonitoringService orchestrates Prometheus, Loki, and k8s queries for per-app monitoring.
type MonitoringService struct {
	prom    *promclient.Client
	loki    *lokiclient.Client
	k8s     ports.KubernetesClient
	appRepo ports.AppRepository
	logs    AppLogSource
	nsFor   func(ctx context.Context, projectID string) string
}

// SetNamespaceResolver reads each app's pods and metrics from its project namespace.
func (s *MonitoringService) SetNamespaceResolver(r func(ctx context.Context, projectID string) string) {
	s.nsFor = r
}

// namespaceOf returns the namespace an app's workloads run in.
func (s *MonitoringService) namespaceOf(ctx context.Context, appID string) string {
	if s.nsFor == nil {
		return appsNamespace
	}
	app, err := s.appRepo.GetApp(ctx, appID)
	if err != nil {
		return appsNamespace
	}
	return s.nsFor(ctx, app.ProjectID)
}

// AppLogSource supplies app logs from the runtime (e.g. Docker) instead of Loki.
type AppLogSource interface {
	Tail(ctx context.Context, app *entities.App, since time.Duration, limit int) ([]dto.MonitoringLogEntry, error)
	Stream(ctx context.Context, app *entities.App, out chan<- dto.MonitoringLogEntry) error
}

// SetLogSource makes app logs come from the runtime (Docker) instead of Loki.
func (s *MonitoringService) SetLogSource(src AppLogSource) { s.logs = src }

func filterLogs(in []dto.MonitoringLogEntry, level, search string) []dto.MonitoringLogEntry {
	out := make([]dto.MonitoringLogEntry, 0, len(in))
	for _, e := range in {
		if search != "" && !strings.Contains(strings.ToLower(e.Line), strings.ToLower(search)) {
			continue
		}
		if level != "" && level != "all" && !strings.Contains(strings.ToLower(e.Line), strings.ToLower(level)) && e.Level != level {
			continue
		}
		out = append(out, e)
	}
	return out
}

// NewMonitoringService creates a new MonitoringService.
func NewMonitoringService(prom *promclient.Client, loki *lokiclient.Client, k8s ports.KubernetesClient, appRepo ports.AppRepository) *MonitoringService {
	return &MonitoringService{prom: prom, loki: loki, k8s: k8s, appRepo: appRepo}
}

// resolveApp looks up the app and verifies ownership, returning the app name (used as pod prefix).
func (s *MonitoringService) resolveApp(ctx context.Context, userID, appID string) (string, error) {
	app, err := s.appRepo.GetApp(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("app not found")
	}
	if app.UserID != userID {
		return "", fmt.Errorf("app not found")
	}
	return app.Name, nil
}

// podSelector returns the k8s label selector for an app's pods.
func podSelector(appName string) string {
	return "zenith.dev/app=" + appName
}

// podRegex returns a regex matching an app's pod names.
func podRegex(appName string) string {
	return appName + "-.*"
}

// GetOverview returns key stats: CPU%, mem, request rate, error rate, p95, pod count.
func (s *MonitoringService) GetOverview(ctx context.Context, userID, appID string) (*dto.MetricsOverview, error) {
	ns := s.namespaceOf(ctx, appID)
	appName, err := s.resolveApp(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	overview := &dto.MetricsOverview{}

	// Pod count + metrics from k8s
	pods, err := s.k8s.ListPods(ctx, ns, podSelector(appName))
	if err == nil {
		overview.PodCount = len(pods)
	}

	podMetrics, err := s.k8s.GetPodMetrics(ctx, ns, podSelector(appName))
	if err == nil && len(podMetrics) > 0 {
		var totalCPU int64
		var totalMem int64
		for _, pm := range podMetrics {
			totalCPU += pm.CPUMillicores
			totalMem += pm.MemoryBytes
		}
		overview.CPUPercent = float64(totalCPU) / 10.0 // millicores to % of 1 core
		overview.MemoryMB = float64(totalMem) / (1024 * 1024)

		// Calculate memory percent from pod resource limits
		var totalMemLimit int64
		for _, p := range pods {
			if p.MemoryLimitBytes > 0 {
				totalMemLimit += p.MemoryLimitBytes
			}
		}
		if totalMemLimit > 0 {
			overview.MemoryPercent = float64(totalMem) / float64(totalMemLimit) * 100
		}
	}

	if s.prom == nil {
		return overview, nil
	}

	// Request rate (APISIX or generic HTTP metrics)
	reqRate, _ := s.prom.QueryInstant(ctx, fmt.Sprintf(
		`sum(rate(apisix_http_status{matched_route=~".*%s.*"}[5m]))`, appName))
	overview.RequestRate = reqRate

	// Error rate
	errRate, _ := s.prom.QueryInstant(ctx, fmt.Sprintf(
		`sum(rate(apisix_http_status{matched_route=~".*%s.*",code=~"5.."}[5m])) / clamp_min(sum(rate(apisix_http_status{matched_route=~".*%s.*"}[5m])), 1) * 100`, appName, appName))
	overview.ErrorRate = errRate

	// P95 latency
	p95, _ := s.prom.QueryInstant(ctx, fmt.Sprintf(
		`histogram_quantile(0.95, sum(rate(apisix_http_latency_bucket{matched_route=~".*%s.*",type="request"}[5m])) by (le)) / 1000`, appName))
	overview.P95Latency = p95

	return overview, nil
}

// timeRangeParams parses a range string to start time and step.
func timeRangeParams(rangeStr string) (start time.Time, step time.Duration) {
	now := time.Now()
	switch rangeStr {
	case "1h":
		return now.Add(-1 * time.Hour), 30 * time.Second
	case "6h":
		return now.Add(-6 * time.Hour), 2 * time.Minute
	case "24h":
		return now.Add(-24 * time.Hour), 5 * time.Minute
	case "7d":
		return now.Add(-7 * 24 * time.Hour), 30 * time.Minute
	default:
		return now.Add(-1 * time.Hour), 30 * time.Second
	}
}

// GetTimeSeries returns a time series for a specific metric.
func (s *MonitoringService) GetTimeSeries(ctx context.Context, userID, appID, metric, timeRange string) (*dto.TimeSeriesResponse, error) {
	ns := s.namespaceOf(ctx, appID)
	appName, err := s.resolveApp(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	if s.prom == nil {
		return &dto.TimeSeriesResponse{Metric: metric, Range: timeRange, Points: []dto.TimeSeriesPoint{}}, nil
	}

	start, step := timeRangeParams(timeRange)
	end := time.Now()

	var promQL string
	regex := podRegex(appName)
	switch metric {
	case "cpu":
		promQL = fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s",container!=""}[5m])) * 100`, ns, regex)
	case "memory":
		promQL = fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s",pod=~"%s",container!=""}) / (1024*1024)`, ns, regex)
	case "requests":
		promQL = fmt.Sprintf(`sum(rate(apisix_http_status{matched_route=~".*%s.*"}[5m])) * 60`, appName)
	case "latency":
		promQL = fmt.Sprintf(`histogram_quantile(0.95, sum(rate(apisix_http_latency_bucket{matched_route=~".*%s.*",type="request"}[5m])) by (le)) / 1000`, appName)
	default:
		return nil, fmt.Errorf("unknown metric: %s", metric)
	}

	points, err := s.prom.QueryRange(ctx, promQL, start, end, step)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	tsPoints := make([]dto.TimeSeriesPoint, 0, len(points))
	for _, p := range points {
		tsPoints = append(tsPoints, dto.TimeSeriesPoint{
			Timestamp: p.Timestamp,
			Value:     p.Value,
		})
	}

	return &dto.TimeSeriesResponse{
		Metric: metric,
		Range:  timeRange,
		Points: tsPoints,
	}, nil
}

// GetLogs queries Loki for log entries from an app.
func (s *MonitoringService) GetLogs(ctx context.Context, userID, appID string, level, search string, limit int, since time.Duration) (*dto.MonitoringLogsResponse, error) {
	ns := s.namespaceOf(ctx, appID)
	appName, err := s.resolveApp(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	if s.logs != nil {
		app, err := s.appRepo.GetApp(ctx, appID)
		if err != nil {
			return nil, fmt.Errorf("app not found")
		}
		if limit <= 0 {
			limit = 100
		}
		if since == 0 {
			since = time.Hour
		}
		entries, err := s.logs.Tail(ctx, app, since, limit)
		if err != nil {
			return nil, err
		}
		entries = filterLogs(entries, level, search)
		return &dto.MonitoringLogsResponse{Entries: entries, Total: len(entries)}, nil
	}

	if s.loki == nil {
		return &dto.MonitoringLogsResponse{Entries: []dto.MonitoringLogEntry{}, Total: 0}, nil
	}

	if limit <= 0 {
		limit = 100
	}
	if since == 0 {
		since = 1 * time.Hour
	}

	// Build LogQL query
	logQL := fmt.Sprintf(`{namespace="%s",app="%s"}`, ns, appName)
	if search != "" {
		logQL += fmt.Sprintf(` |= "%s"`, strings.ReplaceAll(search, `"`, `\"`))
	}
	if level != "" && level != "all" {
		logQL += fmt.Sprintf(` | logfmt | level=~"%s"`, level)
	}

	end := time.Now()
	start := end.Add(-since)

	entries, err := s.loki.QueryRange(ctx, logQL, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("loki query failed: %w", err)
	}

	result := make([]dto.MonitoringLogEntry, 0, len(entries))
	for _, e := range entries {
		lvl := ""
		if v, ok := e.Labels["level"]; ok {
			lvl = v
		}
		result = append(result, dto.MonitoringLogEntry{
			Timestamp: e.Timestamp,
			Line:      e.Line,
			Level:     lvl,
			Labels:    e.Labels,
		})
	}

	return &dto.MonitoringLogsResponse{
		Entries: result,
		Total:   len(result),
	}, nil
}

// StreamLogs sends log entries from Loki tail to the out channel.
func (s *MonitoringService) StreamLogs(ctx context.Context, userID, appID string, out chan<- dto.MonitoringLogEntry) error {
	ns := s.namespaceOf(ctx, appID)
	appName, err := s.resolveApp(ctx, userID, appID)
	if err != nil {
		return err
	}

	if s.logs != nil {
		app, err := s.appRepo.GetApp(ctx, appID)
		if err != nil {
			return fmt.Errorf("app not found")
		}
		return s.logs.Stream(ctx, app, out)
	}

	if s.loki == nil {
		return fmt.Errorf("loki not configured")
	}

	logQL := fmt.Sprintf(`{namespace="%s",app="%s"}`, ns, appName)

	lokiCh := make(chan lokiclient.LogEntry, 100)
	go func() {
		defer close(lokiCh)
		_ = s.loki.Tail(ctx, logQL, lokiCh)
	}()

	for entry := range lokiCh {
		lvl := ""
		if v, ok := entry.Labels["level"]; ok {
			lvl = v
		}
		select {
		case out <- dto.MonitoringLogEntry{
			Timestamp: entry.Timestamp,
			Line:      entry.Line,
			Level:     lvl,
			Labels:    entry.Labels,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// GetAggregatedLogs queries Loki for log entries from multiple apps.
func (s *MonitoringService) GetAggregatedLogs(ctx context.Context, userID string, appIDs []string, level, search string, limit int, since time.Duration) (*dto.MonitoringLogsResponse, error) {
	if s.loki == nil {
		return &dto.MonitoringLogsResponse{Entries: []dto.MonitoringLogEntry{}, Total: 0}, nil
	}

	if limit <= 0 {
		limit = 200
	}
	if since == 0 {
		since = 1 * time.Hour
	}

	// Resolve app names, verify ownership and group them by namespace, so each
	// Loki query only matches the user's own apps in their own namespaces.
	byNS := map[string][]string{}
	for _, id := range appIDs {
		name, err := s.resolveApp(ctx, userID, id)
		if err != nil {
			continue // skip apps the user doesn't own
		}
		ns := s.namespaceOf(ctx, id)
		byNS[ns] = append(byNS[ns], name)
	}
	if len(byNS) == 0 {
		return &dto.MonitoringLogsResponse{Entries: []dto.MonitoringLogEntry{}, Total: 0}, nil
	}

	end := time.Now()
	start := end.Add(-since)

	result := make([]dto.MonitoringLogEntry, 0, limit)
	for ns, names := range byNS {
		entries, err := s.loki.QueryRange(ctx, aggregatedLogQL(ns, names, level, search), start, end, limit)
		if err != nil {
			return nil, fmt.Errorf("loki query failed: %w", err)
		}
		for _, e := range entries {
			result = append(result, dto.MonitoringLogEntry{
				Timestamp: e.Timestamp,
				Line:      e.Line,
				Level:     e.Labels["level"],
				Labels:    e.Labels,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Timestamp.After(result[j].Timestamp) })
	if len(result) > limit {
		result = result[:limit]
	}

	return &dto.MonitoringLogsResponse{
		Entries: result,
		Total:   len(result),
	}, nil
}

var logLevelPattern = regexp.MustCompile(`^[a-z]+$`)

// aggregatedLogQL builds a LogQL query for several apps in one namespace.
// App names are regex-quoted and the search text is a quoted string literal,
// so user input cannot change the query's structure.
func aggregatedLogQL(ns string, appNames []string, level, search string) string {
	quoted := make([]string, len(appNames))
	for i, n := range appNames {
		quoted[i] = regexp.QuoteMeta(n)
	}
	q := fmt.Sprintf(`{namespace=%s,app=~%s}`, strconv.Quote(ns), strconv.Quote(strings.Join(quoted, "|")))
	if search != "" {
		q += " |= " + strconv.Quote(search)
	}
	if level != "" && level != "all" && logLevelPattern.MatchString(level) {
		q += fmt.Sprintf(` | logfmt | level=%s`, strconv.Quote(level))
	}
	return q
}

// GetPods returns pod list with status and resource usage.
func (s *MonitoringService) GetPods(ctx context.Context, userID, appID string) (*dto.PodsResponse, error) {
	ns := s.namespaceOf(ctx, appID)
	appName, err := s.resolveApp(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	selector := podSelector(appName)
	pods, err := s.k8s.ListPods(ctx, ns, selector)
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	// Get metrics
	metricsMap := make(map[string]ports.K8sPodMetrics)
	metrics, _ := s.k8s.GetPodMetrics(ctx, ns, selector)
	for _, m := range metrics {
		metricsMap[m.Name] = m
	}

	result := make([]dto.PodStatus, 0, len(pods))
	for _, p := range pods {
		ps := dto.PodStatus{
			Name:          p.Name,
			Status:        p.Status,
			Ready:         p.Ready,
			Restarts:      p.Restarts,
			StartedAt:     p.StartedAt,
			StatusReason:  p.StatusReason,
			StatusMessage: p.StatusMessage,
			LastExitCode:  p.LastExitCode,
		}
		if m, ok := metricsMap[p.Name]; ok {
			ps.CPUMillicores = m.CPUMillicores
			ps.MemoryMB = float64(m.MemoryBytes) / (1024 * 1024)
		}
		result = append(result, ps)
	}

	return &dto.PodsResponse{Pods: result}, nil
}
