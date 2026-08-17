#!/usr/bin/env bash
set -euo pipefail

required_targets=(
  help test test-unit test-race test-contract lint format-check vet build build-arm64 image-arm64
  run-api run-worker run-local compose-up compose-down
  compose-config smoke coverage validate-static validate clean
  test-component test-component-storage test-component-postgres
  test-component-dynamodb test-component-kafka test-component-kafka-secure
  test-component-secrets
  test-e2e test-e2e-memory test-e2e-ndjson test-e2e-postgres
  test-e2e-dynamodb test-e2e-secrets test-e2e-kafka-secure
  test-cloud-equivalence
  terraform-fmt terraform-init terraform-validate terraform-plan
  k8s-validate test-infrastructure docs-validate
  security security-vuln security-secrets security-config security-image
)

[[ -f Makefile ]] || { echo "missing Makefile" >&2; exit 1; }

help_output="$(make --no-print-directory help)"
for target in "${required_targets[@]}"; do
  grep -Eq "^${target}:" Makefile || { echo "missing Make target: ${target}" >&2; exit 1; }
  grep -Eq "(^|[[:space:]])${target}([[:space:]]|$)" <<<"${help_output}" || {
    echo "Make help does not document: ${target}" >&2
    exit 1
  }
done

for local_default in 'LOCAL_PARTNER_ID ?= local-partner' 'LOCAL_API_KEY ?= local-development-only-key'; do
  grep -Fq "${local_default}" Makefile || {
    echo "missing documented local run default: ${local_default}" >&2
    exit 1
  }
done
grep -Fq 'ENV_FILE ?= .env' Makefile || { echo "Makefile must define the canonical dotenv path" >&2; exit 1; }
grep -Fq 'TFVARS ?=' Makefile || { echo "Makefile must require an explicit Terraform selection file" >&2; exit 1; }
grep -A3 -E '^terraform-plan:' Makefile | grep -Fq 'TFVARS is required' || {
  echo "terraform-plan must reject a missing backend selection" >&2
  exit 1
}
for target in run-api run-worker run-local; do
  grep -A1 -E "^${target}:" Makefile | grep -Fq 'source "$(ENV_FILE)"' || {
    echo "${target} must load the canonical dotenv file first" >&2
    exit 1
  }
done
grep -Eq '^run-api:.*## ' Makefile || { echo "run-api must remain discoverable" >&2; exit 1; }
grep -A1 -E '^run-api:' Makefile | grep -Fq 'PARTNER_ID="$${PARTNER_ID:-$(LOCAL_PARTNER_ID)}" API_KEY="$${API_KEY:-$(LOCAL_API_KEY)}"' || {
  echo "run-api must supply the documented local identity and API key" >&2
  exit 1
}

for template in .env.local.example .env.production.example; do
  [[ -f "${template}" ]] || { echo "missing environment template: ${template}" >&2; exit 1; }
done
grep -Fxq '.env*' .gitignore || { echo ".gitignore must exclude every dotenv variant" >&2; exit 1; }
grep -Fxq '!.env*.example' .gitignore || { echo ".gitignore must retain dotenv example templates" >&2; exit 1; }
for variable in HTTP_ADDRESS PARTNER_ID API_KEY AUTH_MODE TRANSACTION_TYPES DEFAULT_CURRENCY KAFKA_BROKERS KAFKA_TOPIC KAFKA_TLS KAFKA_CA_FILE KAFKA_SASL_USERNAME KAFKA_SASL_PASSWORD STORE_DRIVER STORE_PATH POSTGRES_URL DYNAMODB_ENDPOINT AWS_REGION DYNAMODB_TABLE LOCAL_SECRET_FILE AWS_SECRET_ID KAFKA_TOPIC_PARTITIONS KAFKA_TOPIC_REPLICATION KAFKA_TOPIC_RETENTION; do
  grep -Eq "^${variable}=" .env.local.example .env.production.example || {
    echo "environment templates do not describe: ${variable}" >&2
    exit 1
  }
done

validate_definition="$(grep -E '^validate:' Makefile)"
for dependency in validate-static test-component test-e2e terraform-validate k8s-validate test-infrastructure security; do
  grep -Eq "(^|[[:space:]])${dependency}([[:space:]]|$)" <<<"${validate_definition}" || {
    echo "validate does not require ${dependency}" >&2
    exit 1
  }
done

validate_static_definition="$(grep -E '^validate-static:' Makefile)"
grep -Eq '(^|[[:space:]])test-race([[:space:]]|$)' <<<"${validate_static_definition}" || {
  echo "validate-static does not require test-race" >&2
  exit 1
}

grep -Fq '.SHELLFLAGS := -eu -o pipefail -c' Makefile || {
  echo "Makefile must enable strict shell flags" >&2
  exit 1
}
for binary in transaction-service topic-bootstrap; do
  grep -Fq ".local/bin/${binary}" Makefile || {
    echo "build output is not isolated for: ${binary}" >&2
    exit 1
  }
done
