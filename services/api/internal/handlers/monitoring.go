package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// MonitoringHandler handles the monitoring API endpoints.
type MonitoringHandler struct {
	svc *services.MonitoringService
}

// NewMonitoringHandler creates a new MonitoringHandler.
func NewMonitoringHandler(svc *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{svc: svc}
}

// GetOverview returns key metrics for an app.
// GET /api/v1/apps/:appId/metrics/overview
func (h *MonitoringHandler) GetOverview(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")

	overview, err := h.svc.GetOverview(c.Context(), userID, appID)
	if err != nil {
		return NewNotFound(err.Error())
	}

	return c.JSON(overview)
}

// GetTimeSeries returns time-series data for a specific metric.
// GET /api/v1/apps/:appId/metrics/timeseries?metric=cpu&range=1h
func (h *MonitoringHandler) GetTimeSeries(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")
	metric := c.Query("metric", "cpu")
	timeRange := c.Query("range", "1h")

	ts, err := h.svc.GetTimeSeries(c.Context(), userID, appID, metric, timeRange)
	if err != nil {
		if err.Error() == "app not found" {
			return NewNotFound(err.Error())
		}
		return NewInternal(err.Error())
	}

	return c.JSON(ts)
}

// GetLogs returns log entries from Loki for an app.
// GET /api/v1/apps/:appId/logs?level=error&search=timeout&limit=100&since=1h
func (h *MonitoringHandler) GetLogs(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")
	level := c.Query("level", "")
	search := c.Query("search", "")
	limit := c.QueryInt("limit", 100)

	var since time.Duration
	switch c.Query("since", "1h") {
	case "1h":
		since = 1 * time.Hour
	case "6h":
		since = 6 * time.Hour
	case "24h":
		since = 24 * time.Hour
	case "7d":
		since = 7 * 24 * time.Hour
	default:
		since = 1 * time.Hour
	}

	logs, err := h.svc.GetLogs(c.Context(), userID, appID, level, search, limit, since)
	if err != nil {
		if err.Error() == "app not found" {
			return NewNotFound(err.Error())
		}
		return NewInternal(err.Error())
	}

	return c.JSON(logs)
}

// StreamLogs streams log entries via SSE.
// GET /api/v1/apps/:appId/logs/stream
func (h *MonitoringHandler) StreamLogs(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		logCh := make(chan dto.MonitoringLogEntry, 100)

		ctx := c.Context()
		go func() {
			defer close(logCh)
			_ = h.svc.StreamLogs(ctx, userID, appID, logCh)
		}()

		for {
			select {
			case entry, ok := <-logCh:
				if !ok {
					fmt.Fprintf(w, "event: done\ndata: {}\n\n") //nolint:errcheck
					w.Flush()                                    //nolint:errcheck
					return
				}
				jsonBytes, _ := json.Marshal(entry)
				fmt.Fprintf(w, "data: %s\n\n", jsonBytes) //nolint:errcheck
				w.Flush()                                  //nolint:errcheck

			case <-time.After(30 * time.Second):
				fmt.Fprintf(w, ": keepalive\n\n") //nolint:errcheck
				w.Flush()                          //nolint:errcheck
			}
		}
	})

	return nil
}

// GetPods returns pods with status and resource usage.
// GET /api/v1/apps/:appId/pods
func (h *MonitoringHandler) GetPods(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	appID := c.Params("appId")

	pods, err := h.svc.GetPods(c.Context(), userID, appID)
	if err != nil {
		if err.Error() == "app not found" {
			return NewNotFound(err.Error())
		}
		return NewInternal(err.Error())
	}

	return c.JSON(pods)
}
