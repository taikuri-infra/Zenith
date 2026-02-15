FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o /zenith-api ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S zenith && adduser -S zenith -G zenith

COPY --from=builder /zenith-api /usr/local/bin/zenith-api

USER zenith

EXPOSE 8080

ENTRYPOINT ["zenith-api"]
