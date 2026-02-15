FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w" \
    -o /zenith-operator ./cmd/operator

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S zenith && adduser -S zenith -G zenith

COPY --from=builder /zenith-operator /usr/local/bin/zenith-operator

USER zenith

EXPOSE 8080 8081

ENTRYPOINT ["zenith-operator"]
