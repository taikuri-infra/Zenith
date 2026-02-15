package telemetry

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "zenith-api-http"
	meterName  = "zenith-api-http"
)

// MiddlewareConfig holds optional configuration for the tracing middleware.
type MiddlewareConfig struct {
	// TracerName overrides the default tracer name.
	TracerName string

	// SkipPaths is a list of paths to skip tracing (e.g., /health, /ready).
	SkipPaths []string
}

// Middleware returns a Fiber middleware that instruments HTTP requests with
// OpenTelemetry traces and metrics. Each request gets a span that records
// the HTTP method, path, status code, duration, and any errors.
func Middleware(cfg ...MiddlewareConfig) fiber.Handler {
	var config MiddlewareConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}

	tn := tracerName
	if config.TracerName != "" {
		tn = config.TracerName
	}

	tracer := otel.Tracer(tn)
	meter := otel.Meter(meterName)
	propagator := otel.GetTextMapPropagator()

	skipPathSet := make(map[string]bool, len(config.SkipPaths))
	for _, p := range config.SkipPaths {
		skipPathSet[p] = true
	}

	// Create metric instruments
	requestDuration, _ := meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("Duration of HTTP server requests in milliseconds"),
		metric.WithUnit("ms"),
	)
	requestCount, _ := meter.Int64Counter(
		"http.server.request_count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	activeRequests, _ := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of currently active HTTP requests"),
		metric.WithUnit("{request}"),
	)

	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Skip tracing for configured paths
		if skipPathSet[path] {
			return c.Next()
		}

		// Extract trace context from incoming request headers
		ctx := propagator.Extract(c.UserContext(), requestHeaders(c))

		// Create the span
		spanName := fmt.Sprintf("%s %s", c.Method(), c.Route().Path)
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Method()),
				semconv.HTTPRouteKey.String(c.Route().Path),
				semconv.HTTPTargetKey.String(path),
				semconv.NetHostNameKey.String(c.Hostname()),
				attribute.String("http.user_agent", c.Get("User-Agent")),
			),
		)
		defer span.End()

		// Set the context with the span on the Fiber request
		c.SetUserContext(ctx)

		// Track active requests
		metricAttrs := metric.WithAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.route", c.Route().Path),
		)
		activeRequests.Add(ctx, 1, metricAttrs)
		defer activeRequests.Add(ctx, -1, metricAttrs)

		// Record the start time
		start := time.Now()

		// Process the request
		err := c.Next()

		// Calculate duration
		duration := float64(time.Since(start).Milliseconds())

		// Get status code
		statusCode := c.Response().StatusCode()

		// Add response attributes to span
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(statusCode),
			attribute.Float64("http.duration_ms", duration),
		)

		// Record error if present
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if statusCode >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record metrics
		statusAttrs := metric.WithAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.route", c.Route().Path),
			attribute.Int("http.status_code", statusCode),
		)
		requestDuration.Record(ctx, duration, statusAttrs)
		requestCount.Add(ctx, 1, statusAttrs)

		// Inject trace context into response headers for downstream propagation
		propagator.Inject(ctx, responseHeaders(c))

		return err
	}
}

// requestHeaders adapts Fiber request headers to the propagation.TextMapCarrier interface.
type fiberRequestHeaders struct {
	ctx *fiber.Ctx
}

func requestHeaders(c *fiber.Ctx) propagation.TextMapCarrier {
	return &fiberRequestHeaders{ctx: c}
}

func (h *fiberRequestHeaders) Get(key string) string {
	return h.ctx.Get(key)
}

func (h *fiberRequestHeaders) Set(key string, value string) {
	h.ctx.Request().Header.Set(key, value)
}

func (h *fiberRequestHeaders) Keys() []string {
	var keys []string
	h.ctx.Request().Header.VisitAll(func(k, v []byte) {
		keys = append(keys, string(k))
	})
	return keys
}

// responseHeaders adapts Fiber response headers to the propagation.TextMapCarrier interface.
type fiberResponseHeaders struct {
	ctx *fiber.Ctx
}

func responseHeaders(c *fiber.Ctx) propagation.TextMapCarrier {
	return &fiberResponseHeaders{ctx: c}
}

func (h *fiberResponseHeaders) Get(key string) string {
	return string(h.ctx.Response().Header.Peek(key))
}

func (h *fiberResponseHeaders) Set(key string, value string) {
	h.ctx.Response().Header.Set(key, value)
}

func (h *fiberResponseHeaders) Keys() []string {
	var keys []string
	h.ctx.Response().Header.VisitAll(func(k, v []byte) {
		keys = append(keys, string(k))
	})
	return keys
}
