SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

COMPOSE_PROJECT_NAME ?= emovis-transaction-intake
COMPOSE_FILE ?= compose.yaml

.PHONY: help test test-unit test-race test-contract lint format-check vet build run-api run-worker run-local compose-up compose-down compose-config smoke validate clean

help: ## Show the canonical commands: help test test-unit test-race test-contract lint format-check vet build run-api run-worker run-local compose-up compose-down compose-config smoke validate clean
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: test-unit ## Run all automated Go tests

test-unit: ## Run deterministic unit and contract tests
	go test ./...

test-race: ## Run all Go tests with the race detector
	go test -race ./...

test-contract: ## Run command, API, container, and CI contract tests
	bash tests/commands/makefile_test.sh
	bash tests/contracts/openapi_test.sh
	bash tests/containers/definitions_test.sh
	bash tests/ci/workflow_test.sh

lint: format-check vet ## Run source formatting and static analysis checks

format-check: ## Fail when tracked Go source is not formatted
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet: ## Run Go static analysis
	go vet ./...

build: ## Build every Go command
	mkdir -p .local/bin
	go build -o .local/bin/transaction-service ./cmd/transaction-service
	go build -o .local/bin/topic-bootstrap ./cmd/topic-bootstrap

run-api: ## Run the API process with explicit local configuration
	go run ./cmd/transaction-service api

run-worker: ## Run the outbox worker process with explicit local configuration
	go run ./cmd/transaction-service worker

run-local: ## Run the combined local API and worker process
	go run ./cmd/transaction-service local

compose-up: ## Build and start the complete local system
	docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) up --build --detach --wait

compose-down: ## Stop and remove only this repository's local stack
	docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) down --remove-orphans --volumes

compose-config: ## Validate the complete Compose model without starting containers
	docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) config --quiet

smoke: ## Exercise the documented transaction flow through the local stack
	bash tests/smoke/local.sh

validate: test test-contract lint compose-config ## Run the locally reproducible delivery gates
	env PYTHONDONTWRITEBYTECODE=1 python3 .codex/skills/agent-instruction-hierarchy/scripts/validate_hierarchy.py --root .
	git diff --check

clean: ## Remove only repository-owned build, test, and Compose artifacts
	go clean -testcache
	@if [[ -f $(COMPOSE_FILE) ]]; then docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) down --remove-orphans --volumes; fi
	@find .local tmp coverage -mindepth 1 -delete 2>/dev/null || true
