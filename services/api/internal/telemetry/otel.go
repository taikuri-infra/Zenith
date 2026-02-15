package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Config holds the configuration for the OpenTelemetry SDK.
type Config struct {
	// ServiceName is the name of this service for trace identification.
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// OTLPEndpoint is the gRPC endpoint of the OpenTelemetry Collector.
	// Example: "otel-collector.zenith-system.svc.cluster.local:4317"
	OTLPEndpoint string

	// Environment is the deployment environment (e.g., production, staging).
	Environment string

	// Insecure disables TLS for the OTLP connection (for in-cluster communication).
	Insecure bool

	// SampleRate is the trace sampling rate (0.0 to 1.0). 1.0 samples all traces.
	SampleRate float64
}

// Shutdown is a function returned by Init that should be called on application shutdown
// to flush all pending telemetry data.
type Shutdown func(ctx context.Context) error

// Init initializes the OpenTelemetry SDK with trace and metric providers.
// It configures OTLP exporters that send data to the OpenTelemetry Collector.
// The returned Shutdown function must be called when the application exits.
func Init(ctx context.Context, cfg Config) (Shutdown, error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "zenith-api"
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up trace provider
	traceShutdown, err := setupTraceProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to setup trace provider: %w", err)
	}

	// Set up meter provider
	meterShutdown, err := setupMeterProvider(ctx, cfg, res)
	if err != nil {
		traceShutdown(ctx)
		return nil, fmt.Errorf("failed to setup meter provider: %w", err)
	}

	// Set the global text map propagator for context propagation across services
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := traceShutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if err := meterShutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return fmt.Errorf("shutdown errors: %v", errs)
		}
		return nil
	}

	return shutdown, nil
}

// setupTraceProvider configures and registers the global trace provider.
func setupTraceProvider(ctx context.Context, cfg Config, res *resource.Resource) (Shutdown, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	traceExporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)

	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}

// setupMeterProvider configures and registers the global meter provider.
func setupMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (Shutdown, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExporter,
				sdkmetric.WithInterval(30*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	}, nil
}
