.PHONY: build test lint docker-build run clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

BINARY := zenith-api
DOCKER_IMAGE := zenith-api
DOCKER_TAG := $(VERSION)

build:
	GO111MODULE=on go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/server

run: build
	./bin/$(BINARY)

test:
	GO111MODULE=on go test -v -race -count=1 ./...

test-coverage:
	GO111MODULE=on go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

clean:
	rm -rf bin/ coverage.out coverage.html
