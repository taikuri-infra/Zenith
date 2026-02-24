# Zenith CI/CD
# Usage: make help

# --- Config (override via env or make VAR=value) ---
REGISTRY     ?= registry.stage.freezenith.com/zenith-stage
VERSION      ?= $(shell grep appVersion infra/helm/zenith/Chart.yaml | awk '{print $$2}' | tr -d '"')
PLATFORM     ?= linux/amd64
CHART_DIR    := infra/helm/zenith
CHART_NAME   := zenith
STAGING_HOST ?= 77.42.88.149

IMAGES := zenith-api zenith-landing zenith-mc zenith-mc-demo zenith-web zenith-web-demo zenith-operator

.PHONY: help version \
	test test-api test-web lint lint-api lint-web \
	build build-api build-landing build-mc build-web build-operator \
	push push-api push-landing push-mc push-web push-operator push-all \
	chart-lint chart-package chart-push \
	deploy-staging tf-plan tf-apply \
	ci-images ci-chart ci-terraform ci-all

# --- Help ---
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

version: ## Show current version
	@echo $(VERSION)

# =============================================================================
# Test & Lint
# =============================================================================

test: test-api ## Run all tests

test-api: ## Run Go API tests
	cd services/api && go test ./... -race -count=1

test-web: ## Run web app tests
	pnpm --filter zenith-web test

lint: lint-api ## Run all linters

lint-api: ## Lint Go API
	cd services/api && golangci-lint run ./...

lint-web: ## Lint web apps
	pnpm turbo lint

# =============================================================================
# Docker Build (local, cross-compiled to linux/amd64)
# =============================================================================

build: build-api build-landing build-mc build-web build-operator ## Build all images

build-api: ## Build zenith-api image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-api:$(VERSION) -f services/api/Dockerfile --load .

build-landing: ## Build zenith-landing image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-landing:$(VERSION) -f apps/landing/Dockerfile --load .

build-mc: ## Build zenith-mc + zenith-mc-demo images
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-mc:$(VERSION) -f apps/mission-control/Dockerfile --load .
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-mc-demo:$(VERSION) -f apps/mission-control/Dockerfile --build-arg NEXT_PUBLIC_DEMO_MODE=true --load .

build-web: ## Build zenith-web + zenith-web-demo images
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-web:$(VERSION) -f apps/web/Dockerfile --load .
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-web-demo:$(VERSION) -f apps/web/Dockerfile --build-arg NEXT_PUBLIC_DEMO_MODE=true --load .

build-operator: ## Build zenith-operator image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-operator:$(VERSION) -f services/operator/Dockerfile --load services/operator

# =============================================================================
# Docker Push
# =============================================================================

push-api: ## Push zenith-api to registry
	docker push $(REGISTRY)/zenith-api:$(VERSION)

push-landing: ## Push zenith-landing to registry
	docker push $(REGISTRY)/zenith-landing:$(VERSION)

push-mc: ## Push zenith-mc + demo to registry
	docker push $(REGISTRY)/zenith-mc:$(VERSION)
	docker push $(REGISTRY)/zenith-mc-demo:$(VERSION)

push-web: ## Push zenith-web + demo to registry
	docker push $(REGISTRY)/zenith-web:$(VERSION)
	docker push $(REGISTRY)/zenith-web-demo:$(VERSION)

push-operator: ## Push zenith-operator to registry
	docker push $(REGISTRY)/zenith-operator:$(VERSION)

push-all: push-api push-landing push-mc push-web push-operator ## Push all images

push: build push-all ## Build + push all images

# =============================================================================
# Helm Chart
# =============================================================================

chart-lint: ## Lint Helm chart
	helm lint $(CHART_DIR)

chart-package: ## Package Helm chart
	helm package $(CHART_DIR) -d /tmp

chart-push: chart-package ## Package and push Helm chart to Harbor
	helm push /tmp/$(CHART_NAME)-$(VERSION).tgz oci://$(REGISTRY)

chart: chart-lint chart-push ## Lint + push Helm chart

# =============================================================================
# Deploy
# =============================================================================

tf-plan: ## Terraform plan (staging-k8s)
	cd infra/terraform/staging-k8s && terraform plan

tf-apply: ## Terraform apply (staging-k8s)
	cd infra/terraform/staging-k8s && terraform apply

deploy-staging: push chart-push tf-apply ## Full staging deploy: build, push, apply

# =============================================================================
# CI via act (GitHub Actions local runner)
# =============================================================================

ci-images: ## CI: Build & push Docker images via act
	act -j build-api -j build-landing -j build-mc -j build-web -j build-operator

ci-chart: ## CI: Build & push Helm chart via act
	act -j build-chart

ci-terraform: ## CI: Run Terraform plan via act
	act -j terraform-staging -j terraform-staging-k8s

ci-all: ci-images ci-chart ci-terraform ## CI: Run everything
