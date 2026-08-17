#!/usr/bin/env bash
set -euo pipefail

[[ -f Dockerfile ]] || { echo "missing Dockerfile" >&2; exit 1; }
[[ -f .dockerignore ]] || { echo "missing .dockerignore" >&2; exit 1; }
[[ -f compose.yaml ]] || { echo "missing compose.yaml" >&2; exit 1; }
grep -Fq 'FROM golang:1.26.6-alpine AS build' Dockerfile || { echo "builder must use the remediated Go toolchain" >&2; exit 1; }

for target in api worker local topic-bootstrap; do
  grep -Eq "^FROM .* AS ${target}$" Dockerfile || {
    echo "missing Dockerfile target: ${target}" >&2
    exit 1
  }
done
grep -Eq '^USER [^0]' Dockerfile || { echo "runtime must be non-root" >&2; exit 1; }
for ignored in .git .local tmp coverage; do
  grep -Fxq "${ignored}" .dockerignore || { echo "Docker context does not ignore: ${ignored}" >&2; exit 1; }
done
for ignored in '**/.terraform' '*.tfplan' '*.tfstate' '*.tfstate.*'; do
  grep -Fxq "${ignored}" .dockerignore || { echo "Docker context does not ignore Terraform artifact: ${ignored}" >&2; exit 1; }
done

for service in kafka topic-bootstrap app; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing Compose service: ${service}" >&2
    exit 1
  }
done
for service in postgres dynamodb-local; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing component Compose service: ${service}" >&2
    exit 1
  }
done
for service in e2e-ndjson-init e2e-ndjson e2e-postgres-api e2e-postgres-worker e2e-dynamodb-api e2e-dynamodb-worker e2e-secrets; do
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
