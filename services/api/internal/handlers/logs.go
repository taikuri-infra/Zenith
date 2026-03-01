package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/deploy"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// LogHandler handles deployment log streaming via SSE (Server-Sent Events).
type LogHandler struct {
	appRepo ports.AppRepository
	logHub  *deploy.LogHub
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(appRepo ports.AppRepository, logHub *deploy.LogHub) *LogHandler {
	return &LogHandler{
		appRepo: appRepo,
		logHub:  logHub,
	}
}

// logEntryJSON is the JSON shape for log entries.
type logEntryJSON struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// StreamLogs streams deployment logs via Server-Sent Events (SSE).
// GET /api/v1/apps/:id/deployments/:did/logs
func (h *LogHandler) StreamLogs(c *fiber.Ctx) error {
	appID := c.Params("appId")
	deploymentID := c.Params("did")

	// Verify app exists
	_, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("App not found")
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Subscribe to log stream
	sub := h.logHub.Subscribe(deploymentID, 200)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer sub.Close()

		for {
			select {
			case entry, ok := <-sub.Ch:
				if !ok {
					// Channel closed — deployment finished or cleaned up
					fmt.Fprintf(w, "event: done\ndata: {}\n\n") //nolint:errcheck
					w.Flush()                                    //nolint:errcheck
					return
				}

				data := logEntryJSON{
					Timestamp: entry.Timestamp.Format(time.RFC3339),
					Level:     entry.Level,
					Message:   entry.Message,
				}

				jsonBytes, _ := json.Marshal(data)
				fmt.Fprintf(w, "data: %s\n\n", jsonBytes) //nolint:errcheck
				w.Flush()                                  //nolint:errcheck

			case <-time.After(30 * time.Second):
				// Keep-alive ping
				fmt.Fprintf(w, ": keepalive\n\n") //nolint:errcheck
				w.Flush()                          //nolint:errcheck
			}
		}
	})

	return nil
}

// GetLogs returns stored log history for a deployment (non-streaming).
// GET /api/v1/apps/:id/deployments/:did/logs/history
func (h *LogHandler) GetLogs(c *fiber.Ctx) error {
	appID := c.Params("appId")
	deploymentID := c.Params("did")

	// Verify app exists
	_, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("App not found")
	}

	entries := h.logHub.History(deploymentID)

	result := make([]logEntryJSON, len(entries))
	for i, e := range entries {
		result[i] = logEntryJSON{
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Level:     e.Level,
			Message:   e.Message,
		}
	}

	return c.JSON(fiber.Map{
		"items": result,
		"total": len(result),
	})
}
