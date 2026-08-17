#!/usr/bin/env bash
set -euo pipefail

spec="api/openapi.yaml"
[[ -f "${spec}" ]] || { echo "missing ${spec}" >&2; exit 1; }
grep -Fq 'openapi: 3.0.3' "${spec}"
for path in /ingest/v1/transactions /healthz /readyz /metrics; do
  grep -Fq "${path}:" "${spec}" || { echo "missing path ${path}" >&2; exit 1; }
done
for field in source source_reference transaction_type transaction_time_utc base_amount; do
  grep -Fq "${field}:" "${spec}" || { echo "missing field ${field}" >&2; exit 1; }
done
for status in 200 201 400; do
  grep -Eq "^[[:space:]]+'${status}':" "${spec}" || { echo "missing status ${status}" >&2; exit 1; }
done
go test ./tests/contracts -run TestOpenAPIContractIsStructurallyComplete -count=1
