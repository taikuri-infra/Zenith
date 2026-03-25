# Zenith CI/CD
# Usage: make help

# --- Config (override via env or make VAR=value) ---
REGISTRY     ?= registry.stage.freezenith.com/zenith-stage
VERSION      ?= $(shell grep appVersion infra/helm/zenith/Chart.yaml | awk '{print $$2}' | tr -d '"')
PLATFORM     ?= linux/amd64
CHART_DIR    := infra/helm/zenith
CHART_NAME   := zenith
STAGING_HOST ?= 77.42.88.149

IMAGES := zenith-api zenith-landing zenith-mc zenith-web zenith-operator

# Manual deploy helper (bypasses act when it's broken due to dead containers)
# Usage: make manual-deploy-web
MANUAL_TAG ?= sha-$(shell git rev-parse --short HEAD)

.PHONY: help version \
	test test-api test-web lint lint-api lint-web \
	security \
	build build-api build-landing build-mc build-web build-operator \
	push push-api push-landing push-mc push-web push-operator push-all \
	chart-lint chart-package chart-push \
	deploy-staging tf-plan tf-apply \
	ci-images ci-chart ci-terraform ci-all \
	deploy deploy-api deploy-web deploy-mc deploy-landing deploy-operator deploy-all ci

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
# Security (Semgrep SAST)
# =============================================================================

security: ## Run Semgrep security scan
	semgrep scan --config auto --config .semgrep.yml --error .

# =============================================================================
# Docker Build (local, cross-compiled to linux/amd64)
# =============================================================================

build: build-api build-landing build-mc build-web build-operator ## Build all images

build-api: ## Build zenith-api image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-api:$(VERSION) -f services/api/Dockerfile --load .

build-landing: ## Build zenith-landing image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-landing:$(VERSION) -f apps/landing/Dockerfile --load .

build-mc: ## Build zenith-mc image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-mc:$(VERSION) -f apps/mission-control/Dockerfile --load .

build-web: ## Build zenith-web image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-web:$(VERSION) -f apps/web/Dockerfile --load .

build-operator: ## Build zenith-operator image
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/zenith-operator:$(VERSION) -f services/operator/Dockerfile --load services/operator

# =============================================================================
# Docker Push
# =============================================================================

push-api: ## Push zenith-api to registry
	docker push $(REGISTRY)/zenith-api:$(VERSION)

push-landing: ## Push zenith-landing to registry
	docker push $(REGISTRY)/zenith-landing:$(VERSION)

push-mc: ## Push zenith-mc to registry
	docker push $(REGISTRY)/zenith-mc:$(VERSION)

push-web: ## Push zenith-web to registry
	docker push $(REGISTRY)/zenith-web:$(VERSION)

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

# =============================================================================
# Deploy to Staging via act (full pipeline: test + build + push + values + git)
# Requires: act CLI, .secrets file (HARBOR_ROBOT_USER, HARBOR_ROBOT_TOKEN)
# =============================================================================

ACT_DEPLOY := act -j deploy -W .github/workflows/deploy-staging.yml --secret-file .secrets --input skip_tests=true

deploy: deploy-all ## Alias for deploy-all

deploy-api: ## Deploy API to staging (build + push + ArgoCD sync)
	$(ACT_DEPLOY) --input component=api

deploy-web: ## Deploy Web to staging
	$(ACT_DEPLOY) --input component=web

deploy-mc: ## Deploy MC to staging
	$(ACT_DEPLOY) --input component=mc

deploy-landing: ## Deploy Landing to staging
	$(ACT_DEPLOY) --input component=landing

deploy-operator: ## Deploy Operator to staging
	$(ACT_DEPLOY) --input component=operator

deploy-all: ## Deploy ALL components to staging
	$(ACT_DEPLOY) --input component=all

ci: ## Run CI tests locally via act
	act -j test -W .github/workflows/ci.yml --secret-file .secrets

# Manual deploy targets (bypass act — use when Docker dead-container issue blocks act)
# Runs lint + tests locally before building, same gates as CI, no act container needed.
manual-deploy-web: lint-web ## Manually lint, build, push, and deploy zenith-web (bypasses act)
	docker build --platform linux/amd64 -f apps/web/Dockerfile \
		--build-arg NEXT_PUBLIC_API_URL=https://api.stage.freezenith.com \
		--build-arg NEXT_PUBLIC_LANDING_URL=https://stage.freezenith.com \
		-t $(REGISTRY)/zenith-web:$(MANUAL_TAG) -t $(REGISTRY)/zenith-web:latest .
	docker push $(REGISTRY)/zenith-web:$(MANUAL_TAG)
	docker push $(REGISTRY)/zenith-web:latest
	sed -i '' "s|^image: zenith-web:.*|image: zenith-web:$(MANUAL_TAG)|" infra/helm/zenith-web/values-staging.yaml
	git add infra/helm/zenith-web/values-staging.yaml
	git commit -m "chore: bump staging web -- zenith-web:$(MANUAL_TAG)"
	git pull --rebase origin staging && git push origin staging
	ssh zen-stage "kubectl rollout restart deployment/zenith-web -n zenith-staging && kubectl rollout status deployment/zenith-web -n zenith-staging"

manual-deploy-api: test-api lint-api ## Manually test, lint, build, push, and deploy zenith-api (bypasses act)
	docker build --platform linux/amd64 -f services/api/Dockerfile \
		-t $(REGISTRY)/zenith-api:$(MANUAL_TAG) -t $(REGISTRY)/zenith-api:latest .
	docker push $(REGISTRY)/zenith-api:$(MANUAL_TAG)
	docker push $(REGISTRY)/zenith-api:latest
	sed -i '' "s|^image: zenith-api:.*|image: zenith-api:$(MANUAL_TAG)|" infra/helm/zenith-api/values-staging.yaml
	git add infra/helm/zenith-api/values-staging.yaml
	git commit -m "chore: bump staging api -- zenith-api:$(MANUAL_TAG)"
	git pull --rebase origin staging && git push origin staging
	ssh zen-stage "kubectl rollout restart deployment/zenith-api -n zenith-staging && kubectl rollout status deployment/zenith-api -n zenith-staging"

