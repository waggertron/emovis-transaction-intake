#!/usr/bin/env bash
set -euo pipefail

[[ -f Dockerfile ]] || { echo "missing Dockerfile" >&2; exit 1; }
[[ -f .dockerignore ]] || { echo "missing .dockerignore" >&2; exit 1; }
[[ -f compose.yaml ]] || { echo "missing compose.yaml" >&2; exit 1; }

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

for service in kafka topic-bootstrap app; do
  grep -Eq "^  ${service}:" compose.yaml || {
    echo "missing Compose service: ${service}" >&2
    exit 1
  }
done
grep -Fq 'condition: service_healthy' compose.yaml || { echo "app must wait for healthy dependencies" >&2; exit 1; }
grep -Fq 'condition: service_completed_successfully' compose.yaml || { echo "app must wait for topic bootstrap" >&2; exit 1; }
