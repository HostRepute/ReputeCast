# ReputeCast — common tasks. `make help` lists them.
SHELL := /bin/bash
.DEFAULT_GOAL := help

GO        ?= go
PNPM      ?= pnpm
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse HEAD 2>/dev/null)
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w -X github.com/hostrepute/reputecast/internal/version.Version=$(VERSION) \
             -X github.com/hostrepute/reputecast/internal/version.Commit=$(COMMIT) \
             -X github.com/hostrepute/reputecast/internal/version.Date=$(DATE)
PG_PORT   ?= $(shell grep -s '^PG_PORT=' .env | cut -d= -f2 || true)
PG_PORT   := $(if $(PG_PORT),$(PG_PORT),5432)
DEV_DB    ?= postgres://reputecast:reputecast@localhost:$(PG_PORT)/reputecast?sslmode=disable
TEST_DB   ?= postgres://reputecast:reputecast@localhost:$(PG_PORT)/postgres?sslmode=disable
# Packages CI splits test by test across its shards; each calls testsplit.Main
# from its TestMain.
SPLIT_TESTS := github.com/hostrepute/reputecast/internal/approvals

export REPUTECAST_ENV ?= development

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: deps
deps: ## Install Go and web dependencies
	$(GO) mod download
	cd web && $(PNPM) install --frozen-lockfile

.PHONY: services
services: ## Start Postgres and Mailpit (docker compose)
	docker compose up -d --wait

.PHONY: dev
dev: services ## Run the API (air, live reload) and the web app (Vite) together
	@echo "API  http://localhost:8080   Web  http://localhost:5173   Mail http://localhost:8025"
	@trap 'kill 0' INT TERM; \
	  REPUTECAST_DATABASE__URL="$(DEV_DB)" REPUTECAST_PUBLIC_URL=http://localhost:5173 \
	  REPUTECAST_AUTH__COOKIE_SECURE=false $(GO) tool air & \
	  (cd web && $(PNPM) dev) & \
	  wait

.PHONY: seed
seed: ## Load the Acme demo data into the dev database
	REPUTECAST_DATABASE__URL="$(DEV_DB)" $(GO) run ./cmd/reputecast seed

.PHONY: migrate
migrate: ## Apply migrations to the dev database
	REPUTECAST_DATABASE__URL="$(DEV_DB)" $(GO) run ./cmd/reputecast migrate up

.PHONY: generate
generate: ## Regenerate sqlc code, the OpenAPI document and the web API types
	$(GO) tool sqlc generate
	$(GO) run ./cmd/reputecast openapi > web/openapi.json
	cd web && $(PNPM) api:types

.PHONY: test
test: test-go test-web ## Run all tests

.PHONY: test-go
test-go: ## Go unit and integration tests (uses the compose Postgres when running)
	@if docker compose ps --status running postgres 2>/dev/null | grep -q postgres; then \
	  REPUTECAST_TEST_DATABASE_URL="$(TEST_DB)" $(GO) test -race -timeout 45m ./...; \
	else $(GO) test -race -timeout 45m ./...; fi

.PHONY: test-durations
test-durations: ## Time each Go package's tests, which CI splits its Go jobs by
	@if docker compose ps --status running postgres 2>/dev/null | grep -q postgres; then \
	  export REPUTECAST_TEST_DATABASE_URL="$(TEST_DB)"; fi; \
	$(GO) test -p 2 -race -count=1 -timeout 45m -json ./... | \
	  $(GO) run ./scripts/testshard -record -split $(SPLIT_TESTS) > .github/go-test-durations.txt.new && \
	  mv .github/go-test-durations.txt.new .github/go-test-durations.txt

.PHONY: test-web
test-web: ## Web unit tests (Vitest)
	cd web && $(PNPM) test

.PHONY: e2e
e2e: ## Browser tests (Playwright) against a running dev stack
	cd web && $(PNPM) e2e

.PHONY: lint
lint: lint-go lint-web ## Run all linters

.PHONY: lint-go
lint-go: ## golangci-lint
	@command -v golangci-lint >/dev/null || { echo "install golangci-lint v2: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run ./...

.PHONY: lint-web
lint-web: ## ESLint and TypeScript
	cd web && $(PNPM) lint && $(PNPM) typecheck

.PHONY: web
web: ## Build the web app into web/dist (embedded by go:embed)
	cd web && $(PNPM) build

.PHONY: build
build: web ## Build the single binary into bin/reputecast
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/reputecast ./cmd/reputecast

.PHONY: site
site: ## Check and build the landing site (reputecast.com) into site/dist
	cd site && $(PNPM) install --frozen-lockfile && $(PNPM) lint && $(PNPM) typecheck && $(PNPM) test && $(PNPM) build

.PHONY: site-dev
site-dev: ## Run the landing site with live reload on :5288
	cd site && $(PNPM) install --frozen-lockfile && $(PNPM) dev

.PHONY: sdk
sdk: ## Check, build and test the adapter SDK and template, then run the template through the Go runtime
	cd sdk && $(PNPM) install --frozen-lockfile && $(PNPM) typecheck && $(PNPM) build && $(PNPM) test
	REPUTECAST_ADAPTER_E2E=required $(GO) test -count=1 -run 'TestTemplateAdapterEndToEnd' ./internal/providers/remote/

.PHONY: docker
docker: ## Build the Docker image
	docker build -t reputecast:$(VERSION) --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) .

.PHONY: clean
clean: ## Remove build output
	rm -rf bin tmp web/dist/assets web/dist/index.html site/dist site/.ssr
