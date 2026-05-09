# Terraform OpenSearch Provider - Local Development Makefile
#
# This Makefile replicates the logic from .github/workflows/test.yml so you can
# run the same checks locally before pushing to CI.
#
# Usage:
#   make help                Show all available targets
#   make check               Run fast pre-commit checks (lint + tidy + fmt)
#   make ci-test             Run the full CI test suite for one OS version
#   make ci-test-os2         Run full CI test suite against OpenSearch 2.x
#   make ci-test-os3         Run full CI test suite against OpenSearch 3.x
#
# Environment variables:
#   OS_VERSION               OpenSearch major version to test against (2 or 3)
#   TF_LOG                   Terraform log level (default: INFO)
#   TF_ACC                   Set to 1 to run acceptance tests (default: 1 for test targets)

# ------------------------------------------------------------------------------
# Configuration
# ------------------------------------------------------------------------------

# OpenSearch version to test against (2 or 3)
OS_VERSION ?= 2

# Docker image tag based on version
OSS_IMAGE := opensearchproject/opensearch:$(OS_VERSION)
OPENSEARCH_PREFIX := plugins.security
OPENSEARCH_URL := http://admin:myStrongPassword123%40456@localhost:9200

# Terraform settings
TF_LOG ?= INFO
TF_ACC ?= 1

# Test settings
TEST_PARALLEL := 20
TEST_TIMEOUT := 120m

# Go settings
GO := go
GOPATH := $(shell $(GO) env GOPATH)

# ------------------------------------------------------------------------------
# Detect platform differences
# ------------------------------------------------------------------------------

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	DOCKER_COMPOSE := docker compose
else ifeq ($(UNAME_S),Darwin)
	DOCKER_COMPOSE := docker compose
else
	DOCKER_COMPOSE := docker compose
endif

# ------------------------------------------------------------------------------
# Help target
# ------------------------------------------------------------------------------

.PHONY: help
help: ## Display this help message
	@echo "Terraform OpenSearch Provider - Local Development"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment variables:"
	@echo "  OS_VERSION            OpenSearch version to test (2 or 3, default: $(OS_VERSION))"
	@echo "  TF_LOG                Terraform log level (default: $(TF_LOG))"
	@echo "  TF_ACC                Run acceptance tests (default: $(TF_ACC))"

# ------------------------------------------------------------------------------
# Tool checks
# ------------------------------------------------------------------------------

.PHONY: check-tools
check-tools: ## Verify required tools are installed
	@which $(GO) > /dev/null || (echo "Error: Go is not installed" && exit 1)
	@which terraform > /dev/null || (echo "Error: terraform is not installed" && exit 1)
	@which docker > /dev/null || (echo "Error: docker is not installed" && exit 1)
	@$(DOCKER_COMPOSE) version > /dev/null 2>&1 || (echo "Error: docker compose is not available" && exit 1)
	@echo "All required tools are installed."

.PHONY: check-lint-tool
check-lint-tool: ## Check if golangci-lint v2.x is installed
	@which golangci-lint > /dev/null || (echo "Error: golangci-lint is not installed. Install via:" && echo "  brew install golangci-lint" && exit 1)
	@golangci-lint --version | grep -q "2\." || (echo "Error: golangci-lint v2.x is required. You have:" && golangci-lint --version && echo "Install the correct version via:" && echo "  brew install golangci-lint" && exit 1)

.PHONY: check-goreleaser-tool
check-goreleaser-tool: ## Check if goreleaser is installed
	@which goreleaser > /dev/null || (echo "Error: goreleaser is not installed. Install via:" && echo "  brew install goreleaser" && echo "  OR download from https://github.com/goreleaser/goreleaser/releases" && exit 1)

# ------------------------------------------------------------------------------
# Lint targets
# ------------------------------------------------------------------------------

.PHONY: lint
lint: check-lint-tool ## Run golangci-lint and gofmt checks (same as CI)
	golangci-lint run --verbose --timeout=10m
	@test -z "$$(gofmt -l .)" || (echo "Go code is not formatted. Run 'gofmt -w .' to fix:" && gofmt -l . && exit 1)

# ------------------------------------------------------------------------------
# Format and validation targets
# ------------------------------------------------------------------------------

.PHONY: fmt-check
fmt-check: ## Run terraform fmt -check (same as CI)
	terraform fmt -check -recursive

.PHONY: fmt
fmt: ## Run terraform fmt (fix formatting)
	terraform fmt -recursive

.PHONY: validate
validate: ## Run terraform validate (same as CI)
	terraform validate -no-color

# ------------------------------------------------------------------------------
# Go module targets
# ------------------------------------------------------------------------------

.PHONY: tidy
tidy: ## Run go mod tidy
	$(GO) mod tidy

.PHONY: tidy-check
tidy-check: ## Run go mod tidy and verify no changes (same as CI)
	./script/test-mod-tidy

# ------------------------------------------------------------------------------
# Infrastructure targets (Docker Compose)
# ------------------------------------------------------------------------------

.PHONY: infra-up
infra-up: check-tools ## Start OpenSearch and Dashboards containers
	@UNAME=$$(uname -s); \
	if [ "$$UNAME" = "Linux" ]; then \
		MAX_MAP=$$(sysctl -n vm.max_map_count 2>/dev/null || echo 0); \
		if [ "$$MAX_MAP" -lt 262144 ]; then \
			echo "Error: vm.max_map_count is $$MAX_MAP, but must be >= 262144 for OpenSearch."; \
			echo "Run: sudo sysctl -w vm.max_map_count=262144"; \
			exit 1; \
		fi; \
	fi
	@echo "Starting OpenSearch $(OS_VERSION)..."
	OSS_IMAGE=$(OSS_IMAGE) $(DOCKER_COMPOSE) up --detach
	@echo "Containers started. Run 'make wait' to wait for OpenSearch to be ready."

.PHONY: infra-down
infra-down: ## Stop and remove OpenSearch containers
	@echo "Stopping OpenSearch containers..."
	$(DOCKER_COMPOSE) down
	@echo "Containers stopped."

.PHONY: infra-logs
infra-logs: ## Show logs from OpenSearch containers
	$(DOCKER_COMPOSE) logs -f

.PHONY: infra-ps
infra-ps: ## List running Docker containers
	docker ps -a

# ------------------------------------------------------------------------------
# Wait targets
# ------------------------------------------------------------------------------

.PHONY: wait
wait: ## Wait for OpenSearch to be ready (same as CI)
	@echo "Waiting for OpenSearch at $(OPENSEARCH_URL)..."
	./script/wait-for-endpoint --timeout=60 $(OPENSEARCH_URL)

# ------------------------------------------------------------------------------
# Test targets
# ------------------------------------------------------------------------------

.PHONY: test-unit
test-unit: ## Run unit tests only (no acceptance tests)
	$(GO) test ./... -v -cover -short

.PHONY: test-acc
test-acc: check-tools ## Run acceptance tests (requires 'make infra-up' first)
	@echo "Running acceptance tests against OpenSearch $(OS_VERSION)..."
	export OPENSEARCH_URL=$(OPENSEARCH_URL) && \
	export OPENSEARCH_PREFIX=$(OPENSEARCH_PREFIX) && \
	export TF_LOG=$(TF_LOG) && \
	TF_ACC=$(TF_ACC) $(GO) test ./... -v -parallel $(TEST_PARALLEL) -cover -short -timeout $(TEST_TIMEOUT)

.PHONY: test-acc-os2
test-acc-os2: ## Run acceptance tests against OpenSearch 2.x (auto-manages containers)
	$(MAKE) OS_VERSION=2 infra-up
	$(MAKE) OS_VERSION=2 wait
	$(MAKE) OS_VERSION=2 test-acc || (EXIT_CODE=$$?; $(MAKE) OS_VERSION=2 infra-down; exit $$EXIT_CODE)
	$(MAKE) OS_VERSION=2 infra-down

.PHONY: test-acc-os3
test-acc-os3: ## Run acceptance tests against OpenSearch 3.x (auto-manages containers)
	$(MAKE) OS_VERSION=3 infra-up
	$(MAKE) OS_VERSION=3 wait
	$(MAKE) OS_VERSION=3 test-acc || (EXIT_CODE=$$?; $(MAKE) OS_VERSION=3 infra-down; exit $$EXIT_CODE)
	$(MAKE) OS_VERSION=3 infra-down

# ------------------------------------------------------------------------------
# GoReleaser targets
# ------------------------------------------------------------------------------

.PHONY: goreleaser-check
goreleaser-check: check-goreleaser-tool ## Validate goreleaser configuration (same as CI)
	goreleaser check

# ------------------------------------------------------------------------------
# Full CI simulation targets
# ------------------------------------------------------------------------------

.PHONY: check
check: check-lint-tool tidy-check fmt-check ## Fast pre-commit checks (lint + tidy + fmt)
	@echo "All pre-commit checks passed."

.PHONY: ci-test
ci-test: check-lint-tool tidy-check fmt-check validate ## Run full CI test suite for OS $(OS_VERSION)
	@echo "=== Starting full CI test simulation for OpenSearch $(OS_VERSION) ==="
	$(MAKE) infra-up
	$(MAKE) wait
	$(MAKE) test-acc || (EXIT_CODE=$$?; $(MAKE) infra-down; exit $$EXIT_CODE)
	$(MAKE) infra-down
	@echo "=== Full CI test simulation completed ==="

.PHONY: ci-test-os2
ci-test-os2: ## Run full CI test suite against OpenSearch 2.x
	$(MAKE) OS_VERSION=2 ci-test

.PHONY: ci-test-os3
ci-test-os3: ## Run full CI test suite against OpenSearch 3.x
	$(MAKE) OS_VERSION=3 ci-test

# ------------------------------------------------------------------------------
# Platform-specific notes
# ------------------------------------------------------------------------------
#
# Linux users:
#   The CI workflow sets vm.max_map_count=262144 for OpenSearch.
#   On Linux, run this before 'make infra-up' if needed:
#     sudo sysctl -w vm.max_map_count=262144
#
# macOS users:
#   Docker Desktop handles vm.max_map_count automatically.
#   No manual intervention is needed.
#
# Windows users:
#   Use WSL2 and follow the Linux notes above.
#
# ------------------------------------------------------------------------------
