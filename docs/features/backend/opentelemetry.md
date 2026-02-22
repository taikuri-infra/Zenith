# OpenTelemetry (Backend Module Doc)

## 1. Purpose

Distributed tracing and metrics for the Zenith API. Exports telemetry data via OTLP gRPC to an OpenTelemetry Collector for observability in production.

## 2. Entities

### telemetry.Config

| Field          | Type    | Env Var                        | Default       |
| -------------- | ------- | ------------------------------ | ------------- |
| ServiceName    | string  | —                              | "zenith-api"  |
| ServiceVersion | string  | —                              | from build    |
| OTLPEndpoint   | string  | OTEL_EXPORTER_OTLP_ENDPOINT    | ""            |
| Environment    | string  | ENVIRONMENT                    | "development" |
| Insecure       | bool    | OTEL_INSECURE                  | true          |
| SampleRate     | float64 | OTEL_SAMPLE_RATE               | 1.0           |

## 3. Services (Use Cases)

- `telemetry.Init()` — Initializes trace + metric providers, configures OTLP gRPC exporters, returns Shutdown function
- `telemetry.Middleware()` — Fiber middleware that creates spans per request with HTTP attributes, records duration/count metrics

## 4. Activation

**Opt-in**: Only active when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. Without it, the API runs without tracing overhead.

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 OTEL_INSECURE=true ./zenith-api
```

## 5. Skip Paths

`/health` and `/ready` are excluded from tracing to reduce noise.

## 6. Metrics Collected

- `http.server.duration` — Request duration histogram (ms)
- `http.server.request_count` — Total request counter
- `http.server.active_requests` — Active in-flight requests gauge

## 7. Security Model

- No sensitive data in spans (no request bodies, no auth tokens)
- OTLP connection is in-cluster (insecure by default for service mesh)
- Graceful degradation: if Init fails, API continues without tracing

## 8. Testing Strategy

- `otel_test.go` — 14 unit tests covering config defaults, middleware creation, skip paths, header carriers, status codes
- Integration: deploy with OTel Collector, verify traces in Jaeger/Grafana

## 9. Future Improvements

- Custom span attributes for deployment events
- Database query tracing (pgx instrumentation)
- Frontend RUM (Real User Monitoring) trace propagation
