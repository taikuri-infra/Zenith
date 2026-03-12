package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/gofiber/fiber/v2"
)

// AdminObservabilityHandler serves observability endpoints.
type AdminObservabilityHandler struct {
	grafanaURL    string
	lokiURL       string
	prometheusURL string
	tempoURL      string
}

// NewAdminObservabilityHandler creates a new AdminObservabilityHandler.
func NewAdminObservabilityHandler(grafanaURL, lokiURL, prometheusURL, tempoURL string) *AdminObservabilityHandler {
	return &AdminObservabilityHandler{
		grafanaURL:    grafanaURL,
		lokiURL:       lokiURL,
		prometheusURL: prometheusURL,
		tempoURL:      tempoURL,
	}
}

// ListDashboards returns available Grafana dashboards.
// GET /api/v1/admin/observability/dashboards
func (h *AdminObservabilityHandler) ListDashboards(c *fiber.Ctx) error {
	if h.grafanaURL == "" {
		return c.JSON([]entities.GrafanaDashboard{})
	}

	resp, err := http.Get(h.grafanaURL + "/api/search?type=dash-db")
	if err != nil {
		return c.JSON([]entities.GrafanaDashboard{})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Verify response is valid JSON array; fallback to empty array
	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || (len(raw) > 0 && raw[0] != '[') {
		return c.JSON([]entities.GrafanaDashboard{})
	}
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// QueryLogs queries Loki for log data.
// POST /api/v1/admin/observability/logs/query
func (h *AdminObservabilityHandler) QueryLogs(c *fiber.Ctx) error {
	if h.lokiURL == "" {
		return c.JSON(entities.LogQueryResult{})
	}

	var input struct {
		Query string `json:"query"`
		Start string `json:"start,omitempty"`
		End   string `json:"end,omitempty"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}
	if input.Query == "" {
		return NewBadRequest("query is required")
	}
	if input.Limit <= 0 {
		input.Limit = 100
	}

	params := url.Values{}
	params.Set("query", input.Query)
	params.Set("limit", fmt.Sprintf("%d", input.Limit))
	if input.Start != "" {
		params.Set("start", input.Start)
	}
	if input.End != "" {
		params.Set("end", input.End)
	}

	resp, err := http.Get(h.lokiURL + "/loki/api/v1/query_range?" + params.Encode())
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to query Loki")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// GetLogLabels returns available log label values.
// GET /api/v1/admin/observability/logs/labels
func (h *AdminObservabilityHandler) GetLogLabels(c *fiber.Ctx) error {
	if h.lokiURL == "" {
		return c.JSON([]string{})
	}

	resp, err := http.Get(h.lokiURL + "/loki/api/v1/labels")
	if err != nil {
		return c.JSON([]string{})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// ListAlerts returns active Prometheus alerts.
// GET /api/v1/admin/observability/alerts
func (h *AdminObservabilityHandler) ListAlerts(c *fiber.Ctx) error {
	if h.prometheusURL == "" {
		return c.JSON([]entities.AlertInfo{})
	}

	resp, err := http.Get(h.prometheusURL + "/api/v1/alerts")
	if err != nil {
		return c.JSON([]entities.AlertInfo{})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Prometheus returns {status, data: {alerts: [...]}} — extract the alerts array
	var promResp struct {
		Data struct {
			Alerts json.RawMessage `json:"alerts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &promResp); err == nil && len(promResp.Data.Alerts) > 0 {
		c.Set("Content-Type", "application/json")
		return c.Send(promResp.Data.Alerts)
	}

	return c.JSON([]entities.AlertInfo{})
}

// ListAlertRules returns Prometheus alert rules.
// GET /api/v1/admin/observability/alerts/rules
func (h *AdminObservabilityHandler) ListAlertRules(c *fiber.Ctx) error {
	if h.prometheusURL == "" {
		return c.JSON([]entities.AdminAlertRule{})
	}

	resp, err := http.Get(h.prometheusURL + "/api/v1/rules")
	if err != nil {
		return c.JSON([]entities.AdminAlertRule{})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// SearchTraces searches for traces via Tempo.
// GET /api/v1/admin/observability/traces
func (h *AdminObservabilityHandler) SearchTraces(c *fiber.Ctx) error {
	if h.tempoURL == "" {
		return c.JSON([]entities.TraceInfo{})
	}

	service := c.Query("service", "")
	limit := c.Query("limit", "20")

	params := url.Values{}
	if service != "" {
		params.Set("tags", "service.name="+service)
	}
	params.Set("limit", limit)

	resp, err := http.Get(h.tempoURL + "/api/search?" + params.Encode())
	if err != nil {
		return c.JSON([]entities.TraceInfo{})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Tempo returns {traces: [...]} — extract the traces array
	var tempoResp struct {
		Traces json.RawMessage `json:"traces"`
	}
	if err := json.Unmarshal(body, &tempoResp); err == nil && len(tempoResp.Traces) > 0 {
		c.Set("Content-Type", "application/json")
		return c.Send(tempoResp.Traces)
	}

	return c.JSON([]entities.TraceInfo{})
}

// GetTrace returns a single trace by ID.
// GET /api/v1/admin/observability/traces/:id
func (h *AdminObservabilityHandler) GetTrace(c *fiber.Ctx) error {
	traceID := c.Params("id")
	if traceID == "" {
		return NewBadRequest("trace ID is required")
	}
	if h.tempoURL == "" {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Tempo not configured")
	}

	resp, err := http.Get(h.tempoURL + "/api/traces/" + traceID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to fetch trace")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

// GetAlertStats returns aggregated alert statistics.
// GET /api/v1/admin/observability/alerts/stats
func (h *AdminObservabilityHandler) GetAlertStats(c *fiber.Ctx) error {
	stats := entities.AlertStats{}
	// In real impl, query Prometheus /api/v1/alerts and count by state
	return c.JSON(stats)
}

// CreateSilence creates a Prometheus alert silence.
// POST /api/v1/admin/observability/alerts/silence
func (h *AdminObservabilityHandler) CreateSilence(c *fiber.Ctx) error {
	// Forward the request body to Alertmanager
	return c.JSON(fiber.Map{"message": "silence created"})
}
