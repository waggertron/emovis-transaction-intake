#!/usr/bin/env bash
set -euo pipefail

[[ -f Dockerfile ]] || { echo "missing Dockerfile" >&2; exit 1; }
[[ -f .dockerignore ]] || { echo "missing .dockerignore" >&2; exit 1; }
[[ -f compose.yaml ]] || { echo "missing compose.yaml" >&2; exit 1; }
grep -Fq 'golang:1.26.6-alpine AS build' Dockerfile || { echo "builder must use the remediated Go toolchain" >&2; exit 1; }

for target in api worker local topic-bootstrap; do
  grep -Eq "^FROM .* AS ${target}$" Dockerfile || {
    echo "missing Dockerfile target: ${target}" >&2
    exit 1
  }
done
grep -Eq '^USER [^0]' Dockerfile || { echo "runtime must be non-root" >&2; exit 1; }
grep -Fq 'build-arm64:' Makefile || { echo "missing Linux ARM64 artifact validation target for Graviton" >&2; exit 1; }
grep -Fq 'GOOS=linux GOARCH=arm64' Makefile || { echo "ARM64 target must compile for Linux ARM64" >&2; exit 1; }
grep -Fq 'make build-arm64' .github/workflows/ci.yml || { echo "CI does not validate the Graviton artifact architecture" >&2; exit 1; }
grep -Fq 'image-arm64:' Makefile || { echo "missing Linux ARM64 production image validation target" >&2; exit 1; }
grep -Fq 'make image-arm64' .github/workflows/ci.yml || { echo "CI does not validate the Graviton image architecture" >&2; exit 1; }
grep -Fq 'ARG TARGETARCH' Dockerfile || { echo "Dockerfile does not honor the target architecture" >&2; exit 1; }
grep -Fq 'down --remove-orphans --volumes >/dev/null' tests/smoke/local.sh || { echo "smoke must clear stale project state before startup" >&2; exit 1; }
grep -B1 -F 'make --no-print-directory compose-up' tests/smoke/local.sh | grep -Fq 'cleanup_stack' || { echo "smoke must clear stale stack immediately before startup" >&2; exit 1; }
for ignored in .git .local tmp coverage; do
  grep -Fxq "${ignored}" .dockerignore || { echo "Docker context does not ignore: ${ignored}" >&2; exit 1; }
done
for ignored in '**/.terraform' '*.tfplan' '*.tfstate' '*.tfstate.*'; do
  grep -Fxq "${ignored}" .dockerignore || { echo "Docker context does not ignore Terraform artifact: ${ignored}" >&2; exit 1; }
done

for service in kafka topic-bootstrap app-data-init app; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing Compose service: ${service}" >&2
    exit 1
  }
done
grep -A24 -E '^  app:' compose.yaml | grep -Fq 'STORE_DRIVER: ndjson' || { echo "default app must use durable NDJSON storage" >&2; exit 1; }
grep -A24 -E '^  app:' compose.yaml | grep -Fq 'STORE_PATH: /data/transactions.ndjson' || { echo "default app NDJSON path missing" >&2; exit 1; }
grep -A24 -E '^  app:' compose.yaml | grep -Fq 'app-data:/data' || { echo "default app durable volume missing" >&2; exit 1; }
grep -A12 -E '^  app-data-init:' compose.yaml | grep -Fq 'cap_add: [CHOWN]' || { echo "default app volume init must use narrow CHOWN capability" >&2; exit 1; }
grep -Eq '^  app-data:$' compose.yaml || { echo "default app named volume missing" >&2; exit 1; }
for service in postgres dynamodb-local; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing component Compose service: ${service}" >&2
    exit 1
  }
done
for service in e2e-ndjson-init e2e-secrets-init e2e-ndjson e2e-postgres-api e2e-postgres-worker e2e-dynamodb-api e2e-dynamodb-worker e2e-secrets; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing end-to-end Compose service: ${service}" >&2
    exit 1
  }
done
for service in kafka-secure-init kafka-secure topic-bootstrap-secure e2e-kafka-secure; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing secure-Kafka Compose service: ${service}" >&2
    exit 1
  }
done
grep -Fq 'profiles: [component-postgres, e2e-postgres]' compose.yaml || { echo "PostgreSQL component/E2E profiles missing" >&2; exit 1; }
grep -Fq 'profiles: [component-dynamodb, e2e-dynamodb]' compose.yaml || { echo "DynamoDB component/E2E profiles missing" >&2; exit 1; }
for profile in e2e-ndjson e2e-postgres e2e-dynamodb; do
  grep -Fq "${profile}" compose.yaml || { echo "end-to-end profile missing: ${profile}" >&2; exit 1; }
done
grep -Fq 'condition: service_healthy' compose.yaml || { echo "app must wait for healthy dependencies" >&2; exit 1; }
grep -Fq 'condition: service_completed_successfully' compose.yaml || { echo "app must wait for topic bootstrap" >&2; exit 1; }
grep -Fq 'user: "${E2E_SECRET_UID:-65532}:${E2E_SECRET_GID:-65532}"' compose.yaml || {
  echo "secret-provider E2E service must use the host fixture owner without root" >&2
  exit 1
}
grep -Fq 'export E2E_SECRET_UID="$(id -u)"' tests/e2e/secrets.sh || {
  echo "secret-provider E2E must export the fixture owner UID" >&2
  exit 1
}
grep -Fq 'export E2E_SECRET_GID="$(id -g)"' tests/e2e/secrets.sh || {
  echo "secret-provider E2E must export the fixture owner GID" >&2
  exit 1
}
grep -Fq 'command: ["chown", "${E2E_SECRET_UID:-65532}:${E2E_SECRET_GID:-65532}", "/data"]' compose.yaml || {
  echo "secret-provider E2E volume must be initialized for the fixture owner" >&2
  exit 1
}
