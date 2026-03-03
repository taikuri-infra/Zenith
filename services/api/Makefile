.PHONY: build test lint docker-build run clean stage-version stage-health stage-status stage-logs e2e-infra e2e-smoke e2e

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

# --- Staging helpers ---
STAGE_API := https://api.stage.freezenith.com

stage-version:
	@curl -s $(STAGE_API)/api/v1/version | jq .

stage-health:
	@curl -s $(STAGE_API)/health | jq .

stage-status:
	@echo "=== Pod ==="
	@ssh zen-stage "kubectl get pod -n zenith-staging -l app=zenith-api -o wide"
	@echo "\n=== Image ==="
	@ssh zen-stage "kubectl get pod -n zenith-staging -l app=zenith-api -o jsonpath='{.items[0].spec.containers[0].image}'"
	@echo "\n\n=== ArgoCD ==="
	@ssh zen-stage "kubectl get application zenith-api -n argocd -o jsonpath='sync={.status.sync.status} health={.status.health.status} rev={.status.sync.revision}'"
	@echo ""

stage-logs:
	@ssh zen-stage "kubectl logs -n zenith-staging -l app=zenith-api --tail=50"

stage-restart:
	@ssh zen-stage "kubectl rollout restart deployment/zenith-api -n zenith-staging"
	@ssh zen-stage "kubectl rollout status deployment/zenith-api -n zenith-staging --timeout=120s"

# --- E2E tests ---
e2e-infra:
	@python3 ../../infra/scripts/e2e-infra-staging.py

e2e-smoke:
	@python3 ../../infra/scripts/e2e-smoke-staging.py --base-url $(STAGE_API) --no-port-forward

e2e: e2e-infra e2e-smoke
