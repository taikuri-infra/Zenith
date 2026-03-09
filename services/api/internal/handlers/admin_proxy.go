package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AdminProxyHandler provides authenticated reverse proxy to admin services.
type AdminProxyHandler struct {
	targets map[string]string
}

// NewAdminProxyHandler creates a new AdminProxyHandler with service target URLs.
func NewAdminProxyHandler(targets map[string]string) *AdminProxyHandler {
	return &AdminProxyHandler{targets: targets}
}

// Proxy handles reverse proxy requests to admin services.
// ALL /api/v1/admin/proxy/:service/*
func (h *AdminProxyHandler) Proxy(c *fiber.Ctx) error {
	service := c.Params("service")
	if service == "" {
		return NewBadRequest("service is required")
	}

	targetBase, ok := h.targets[service]
	if !ok {
		return NewNotFound("proxy target")
	}

	// Build target URL
	path := c.Params("*")
	targetURL := targetBase
	if path != "" {
		targetURL = strings.TrimRight(targetBase, "/") + "/" + path
	}
	if q := string(c.Request().URI().QueryString()); q != "" {
		targetURL += "?" + q
	}

	// Create upstream request
	req, err := http.NewRequestWithContext(c.Context(), c.Method(), targetURL, strings.NewReader(string(c.Body())))
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to create proxy request")
	}

	// Forward headers (except Authorization which is for our API)
	for k, v := range map[string]string{
		"Content-Type": c.Get("Content-Type"),
		"Accept":       c.Get("Accept"),
	} {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "upstream service unavailable")
	}
	defer resp.Body.Close()

	// Forward response headers
	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Set(k, v)
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return c.Status(resp.StatusCode).Send(body)
}
