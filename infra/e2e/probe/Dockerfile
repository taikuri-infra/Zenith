# e2e-probe — the sample app the E2E suite deploys (see main.go).
FROM golang:1.25-alpine AS build
# Optional module proxy for restricted networks (run.sh passes GOPROXY/GOSUMDB through).
ARG GOPROXY
ARG GOSUMDB
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /probe .

# Runtime from Docker Hub (a registry mirror can serve it); gcr.io/distroless is not
# reachable from every network. Static binary, unprivileged user.
FROM alpine:3.21
RUN adduser -D -u 65532 -g nonroot nonroot
COPY --from=build /probe /probe
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/probe"]
