# ---- Builder ----
# Run the builder natively on the host arch and cross-compile to the target
# arch (fast — no QEMU emulation for the Go build).
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Copy dependency files first for layer caching
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY services/api/ .

# Build arguments for version injection
ARG VERSION=dev
ARG BUILD_TIME=unknown
# TARGETARCH is provided automatically by buildx (amd64 / arm64) — cross-compile to it.
ARG TARGETARCH
# Build a statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o /zenith-api ./cmd/server

# ---- Runner ----
FROM alpine:3.21

LABEL org.opencontainers.image.title="zenith-api"
LABEL org.opencontainers.image.description="Zenith API Server - Go backend"
LABEL org.opencontainers.image.source="https://github.com/dotechhq/zenith"
LABEL org.opencontainers.image.vendor="DoTech"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata git

# Create non-root user
RUN addgroup -S zenith && adduser -S zenith -G zenith

# Copy the binary from builder
COPY --from=builder /zenith-api /usr/local/bin/zenith-api

USER zenith

EXPOSE 8080

ENTRYPOINT ["zenith-api"]
