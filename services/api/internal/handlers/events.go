package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/gofiber/fiber/v2"
)

// EventHandler handles real-time deployment event streaming via SSE.
type EventHandler struct {
	eventHub *deploy.EventHub
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(eventHub *deploy.EventHub) *EventHandler {
	return &EventHandler{
		eventHub: eventHub,
	}
}

// eventJSON is the JSON shape for SSE events.
type eventJSON struct {
	Type         string `json:"type"`
	AppID        string `json:"app_id"`
	AppName      string `json:"app_name"`
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
	Image        string `json:"image,omitempty"`
	Message      string `json:"message,omitempty"`
	Timestamp    string `json:"timestamp"`
}

// StreamEvents streams deployment events via Server-Sent Events (SSE).
// GET /api/v1/events
func (h *EventHandler) StreamEvents(c *fiber.Ctx) error {
	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	// Subscribe to event stream
	sub := h.eventHub.Subscribe(50)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer sub.Close()

		for {
			select {
			case event, ok := <-sub.Ch:
				if !ok {
					return
				}

				data := eventJSON{
					Type:         string(event.Type),
					AppID:        event.AppID,
					AppName:      event.AppName,
					DeploymentID: event.DeploymentID,
					Status:       event.Status,
					Image:        event.Image,
					Message:      event.Message,
					Timestamp:    event.Timestamp.Format(time.RFC3339),
				}

				jsonBytes, _ := json.Marshal(data)
				fmt.Fprintf(w, "event: deploy\ndata: %s\n\n", jsonBytes) //nolint:errcheck
				w.Flush()                                                 //nolint:errcheck

			case <-time.After(30 * time.Second):
				// Keep-alive ping
				fmt.Fprintf(w, ": keepalive\n\n") //nolint:errcheck
				w.Flush()                          //nolint:errcheck
			}
		}
	})

	return nil
}

// GetRecentEvents returns stored event history (non-streaming).
// GET /api/v1/events/history
func (h *EventHandler) GetRecentEvents(c *fiber.Ctx) error {
	events := h.eventHub.History()

	result := make([]eventJSON, len(events))
	for i, e := range events {
		result[i] = eventJSON{
			Type:         string(e.Type),
			AppID:        e.AppID,
			AppName:      e.AppName,
			DeploymentID: e.DeploymentID,
			Status:       e.Status,
			Image:        e.Image,
			Message:      e.Message,
			Timestamp:    e.Timestamp.Format(time.RFC3339),
		}
	}

	return c.JSON(fiber.Map{
		"items": result,
		"total": len(result),
	})
}
