SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

COMPOSE_PROJECT_NAME ?= emovis-transaction-intake
COMPOSE_FILE ?= compose.yaml
GITLEAKS_IMAGE ?= zricethezav/gitleaks:v8.24.2
TRIVY_IMAGE ?= aquasec/trivy:0.59.1
SECURITY_IMAGE ?= emovis-transaction-intake:security-review
ARM64_IMAGE ?= emovis-transaction-intake:arm64-validation
ENV_FILE ?= .env
TFVARS ?=
LOCAL_PARTNER_ID ?= local-partner
LOCAL_API_KEY ?= local-development-only-key
LOCAL_AUTH_MODE ?= disabled
LOCAL_TRANSACTION_TYPES ?= toll
LOCAL_DEFAULT_CURRENCY ?= USD

.PHONY: help test test-unit test-race test-contract lint format-check vet build build-arm64 image-arm64 run-api run-worker run-local compose-up compose-down compose-config smoke coverage test-component test-component-storage test-component-postgres test-component-dynamodb test-component-kafka test-component-kafka-secure test-component-secrets test-e2e test-e2e-memory test-e2e-ndjson test-e2e-postgres test-e2e-dynamodb test-e2e-secrets test-e2e-kafka-secure test-cloud-equivalence terraform-fmt terraform-init terraform-validate terraform-plan k8s-validate test-infrastructure docs-validate security security-vuln security-secrets security-config security-image validate-static validate clean

help: ## Show all canonical test, build, run, Compose, component, validation, and cleanup commands
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
	bash tests/coverage/coverage_gate_test.sh
	cd .codex/skills/agent-instruction-hierarchy/scripts && env PYTHONDONTWRITEBYTECODE=1 python3 -m unittest validate_hierarchy_test.py

lint: format-check vet ## Run source formatting and static analysis checks

format-check: ## Fail when tracked Go source is not formatted
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet: ## Run Go static analysis
	go vet ./...

build: ## Build every Go command
	mkdir -p .local/bin
	go build -o .local/bin/transaction-service ./cmd/transaction-service
	go build -o .local/bin/topic-bootstrap ./cmd/topic-bootstrap

build-arm64: ## Cross-compile every production command for the Graviton EKS node architecture
	mkdir -p .local/bin/linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o .local/bin/linux-arm64/transaction-service ./cmd/transaction-service
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o .local/bin/linux-arm64/topic-bootstrap ./cmd/topic-bootstrap
	file .local/bin/linux-arm64/transaction-service .local/bin/linux-arm64/topic-bootstrap | grep -q 'ARM aarch64'

image-arm64: ## Build and inspect the production API image for Linux ARM64
	docker buildx build --platform linux/arm64 --target api --load -t $(ARM64_IMAGE) .
	docker image inspect --format '{{.Architecture}}' $(ARM64_IMAGE) | grep -qx arm64

run-api: ## Run the API process with documented non-production local credentials
	@if [[ -f "$(ENV_FILE)" ]]; then set -a; source "$(ENV_FILE)"; set +a; fi; AUTH_MODE="$${AUTH_MODE:-$(LOCAL_AUTH_MODE)}" TRANSACTION_TYPES="$${TRANSACTION_TYPES:-$(LOCAL_TRANSACTION_TYPES)}" DEFAULT_CURRENCY="$${DEFAULT_CURRENCY:-$(LOCAL_DEFAULT_CURRENCY)}" PARTNER_ID="$${PARTNER_ID:-$(LOCAL_PARTNER_ID)}" API_KEY="$${API_KEY:-$(LOCAL_API_KEY)}" go run ./cmd/transaction-service api

run-worker: ## Run the worker with documented non-production local credentials
	@if [[ -f "$(ENV_FILE)" ]]; then set -a; source "$(ENV_FILE)"; set +a; fi; AUTH_MODE="$${AUTH_MODE:-$(LOCAL_AUTH_MODE)}" TRANSACTION_TYPES="$${TRANSACTION_TYPES:-$(LOCAL_TRANSACTION_TYPES)}" DEFAULT_CURRENCY="$${DEFAULT_CURRENCY:-$(LOCAL_DEFAULT_CURRENCY)}" PARTNER_ID="$${PARTNER_ID:-$(LOCAL_PARTNER_ID)}" API_KEY="$${API_KEY:-$(LOCAL_API_KEY)}" go run ./cmd/transaction-service worker

run-local: ## Run the combined local API and worker process
	@if [[ -f "$(ENV_FILE)" ]]; then set -a; source "$(ENV_FILE)"; set +a; fi; AUTH_MODE="$${AUTH_MODE:-$(LOCAL_AUTH_MODE)}" TRANSACTION_TYPES="$${TRANSACTION_TYPES:-$(LOCAL_TRANSACTION_TYPES)}" DEFAULT_CURRENCY="$${DEFAULT_CURRENCY:-$(LOCAL_DEFAULT_CURRENCY)}" PARTNER_ID="$${PARTNER_ID:-$(LOCAL_PARTNER_ID)}" API_KEY="$${API_KEY:-$(LOCAL_API_KEY)}" go run ./cmd/transaction-service local

compose-up: ## Build and start the complete local system
	docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) up --build --detach --wait

compose-down: ## Stop and remove only this repository's local stack
	docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) down --remove-orphans --volumes

compose-config: ## Validate the complete Compose model without starting containers
	docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) config --quiet

smoke: ## Exercise the documented transaction flow through the local stack
	bash tests/smoke/local.sh

coverage: ## Enforce at least 85 percent statement coverage in every production Go package
	mkdir -p .local/coverage
	go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./... | sed '/^$$/d' >.local/coverage/expected-packages.txt
	go test -coverprofile=.local/coverage/unit.out ./... | tee .local/coverage/packages.txt
	bash tests/coverage/check.sh .local/coverage/packages.txt .local/coverage/expected-packages.txt 85

test-component-postgres: ## Run the shared store contract against Compose PostgreSQL
	bash tests/component/postgres.sh

test-component-dynamodb: ## Run the shared store contract against DynamoDB Local
	bash tests/component/dynamodb.sh

test-component-kafka: ## Run production Kafka behavior against the local plaintext broker
	bash tests/component/kafka.sh

test-component-kafka-secure: ## Run production Kafka behavior against local TLS and SASL/SCRAM
	bash tests/component/kafka_secure.sh

test-component-secrets: ## Run the secret-provider contract against its local substitute
	bash tests/component/secrets.sh

test-component-storage: test-component-postgres test-component-dynamodb ## Run every real local storage component contract

test-component: test-component-storage test-component-kafka test-component-kafka-secure test-component-secrets test-cloud-equivalence ## Run every real local external-service component test

test-e2e-memory: ## Run the production HTTP, memory, outbox, and Kafka path
	bash tests/e2e/memory.sh

test-e2e-ndjson: ## Run the production HTTP, persistent NDJSON, outbox, and Kafka path
	bash tests/e2e/ndjson.sh

test-e2e-postgres: ## Run separate production API and worker processes with PostgreSQL and Kafka
	bash tests/e2e/postgres.sh

test-e2e-dynamodb: ## Run separate production API and worker processes with DynamoDB Local and Kafka
	bash tests/e2e/dynamodb.sh

test-e2e-secrets: ## Run the production processes with the local secret-provider implementation
	bash tests/e2e/secrets.sh

test-e2e-kafka-secure: ## Run the production path with local TLS and SASL/SCRAM Kafka
	bash tests/e2e/kafka_secure.sh

test-e2e: test-e2e-memory test-e2e-ndjson test-e2e-postgres test-e2e-dynamodb test-e2e-secrets test-e2e-kafka-secure ## Run every local implementation end to end

test-cloud-equivalence: test-e2e-postgres test-e2e-dynamodb test-e2e-secrets test-e2e-kafka-secure ## Prove each selected cloud boundary through its production local equivalent

terraform-fmt: ## Check Terraform formatting without changing files
	terraform -chdir=infra/terraform fmt -check -recursive

terraform-init: ## Initialize pinned Terraform providers without a backend
	terraform -chdir=infra/terraform init -backend=false -input=false

terraform-validate: terraform-init ## Validate Terraform without cloud credentials or apply
	terraform -chdir=infra/terraform validate

terraform-plan: terraform-init ## Create a no-credential, non-applying example Terraform plan
	@test -n "$(TFVARS)" || { echo "TFVARS is required; choose dynamodb.tfvars.example or postgres.tfvars.example" >&2; exit 2; }
	mkdir -p .local/terraform
	terraform -chdir=infra/terraform plan -input=false -refresh=false -lock=false -var-file=$(TFVARS) -out=../../.local/terraform/example.tfplan

k8s-validate: ## Validate Kubernetes YAML structure and client-side schemas without applying
	kubectl kustomize deploy/kubernetes | env PYTHONDONTWRITEBYTECODE=1 python3 tests/infrastructure/validate_kubernetes.py

test-infrastructure: terraform-init ## Run local Terraform and Kubernetes security-policy and selection contracts
	bash tests/infrastructure/infrastructure_test.sh
	bash tests/infrastructure/storage_selection.sh

docs-validate: ## Validate all repository-relative Markdown links
	env PYTHONDONTWRITEBYTECODE=1 python3 tests/documentation/link_check.py .

security-vuln: ## Scan reachable Go code for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

security-secrets: ## Scan the working tree and Git history for secrets
	docker run --rm -v $(CURDIR):/repo $(GITLEAKS_IMAGE) dir --no-banner --redact --config /repo/.gitleaks.toml /repo
	docker run --rm -v $(CURDIR):/repo $(GITLEAKS_IMAGE) git --no-banner --redact --config /repo/.gitleaks.toml /repo

security-config: ## Scan source, containers, Kubernetes, and Terraform for severe findings
	docker run --rm -v $(CURDIR):/repo $(TRIVY_IMAGE) fs --cache-dir /tmp/trivy --skip-check-update --scanners vuln,misconfig,secret --severity HIGH,CRITICAL --exit-code 1 --skip-dirs .git --skip-dirs .local /repo

security-image: ## Build and scan the production API image for severe vulnerabilities
	docker build --target api -t $(SECURITY_IMAGE) .
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(TRIVY_IMAGE) image --cache-dir /tmp/trivy --severity HIGH,CRITICAL --exit-code 1 $(SECURITY_IMAGE)

security: security-vuln security-secrets security-config security-image ## Run all locally reproducible security gates

validate-static: test test-race test-contract coverage lint build-arm64 image-arm64 compose-config docs-validate ## Run fast locally reproducible delivery gates
	env PYTHONDONTWRITEBYTECODE=1 python3 .codex/skills/agent-instruction-hierarchy/scripts/validate_hierarchy.py --root .
	git diff --check

validate: validate-static test-component test-e2e terraform-fmt terraform-validate k8s-validate test-infrastructure security ## Run every local delivery gate, including real substitutes

clean: ## Remove only repository-owned build, test, and Compose artifacts
	go clean -testcache
	@if [[ -f $(COMPOSE_FILE) ]]; then docker compose -p $(COMPOSE_PROJECT_NAME) -f $(COMPOSE_FILE) down --remove-orphans --volumes; fi
	@find .local tmp coverage -mindepth 1 -delete 2>/dev/null || true
