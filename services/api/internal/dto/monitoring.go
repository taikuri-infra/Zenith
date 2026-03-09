package dto

import "time"

// MetricsOverview holds key stats for an app.
type MetricsOverview struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      float64 `json:"memory_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	RequestRate   float64 `json:"request_rate"`
	ErrorRate     float64 `json:"error_rate"`
	P95Latency    float64 `json:"p95_latency_ms"`
	PodCount      int     `json:"pod_count"`
}

// TimeSeriesPoint represents a single data point in a time series.
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TimeSeriesResponse holds a metric time series.
type TimeSeriesResponse struct {
	Metric string            `json:"metric"`
	Range  string            `json:"range"`
	Points []TimeSeriesPoint `json:"points"`
}

// MonitoringLogEntry represents a log entry from Loki.
type MonitoringLogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Line      string            `json:"line"`
	Level     string            `json:"level,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// MonitoringLogsResponse holds log query results.
type MonitoringLogsResponse struct {
	Entries []MonitoringLogEntry `json:"entries"`
	Total   int                  `json:"total"`
}

// PodStatus holds combined pod info and metrics.
type PodStatus struct {
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	Ready         bool      `json:"ready"`
	Restarts      int32     `json:"restarts"`
	CPUMillicores int64     `json:"cpu_millicores"`
	MemoryMB      float64   `json:"memory_mb"`
	StartedAt     time.Time `json:"started_at"`
	StatusReason  string    `json:"status_reason,omitempty"`
	StatusMessage string    `json:"status_message,omitempty"`
	LastExitCode  int32     `json:"last_exit_code,omitempty"`
}

// PodsResponse holds the list of pods for an app.
type PodsResponse struct {
	Pods []PodStatus `json:"pods"`
}
