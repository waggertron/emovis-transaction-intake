#!/usr/bin/env bash
set -euo pipefail

spec="api/openapi.yaml"
[[ -f "${spec}" ]] || { echo "missing ${spec}" >&2; exit 1; }
grep -Fq 'openapi: 3.0.3' "${spec}"
cmp -s "${spec}" "docs/2026-08-18-supplied-openapi.yaml" || { echo "active contract differs from supplied contract" >&2; exit 1; }
grep -Fq '/ingest/v1/transactions:' "${spec}" || { echo "missing ingest path" >&2; exit 1; }
for field in source source_reference transaction_type transaction_time_utc base_amount; do
  grep -Fq "${field}:" "${spec}" || { echo "missing field ${field}" >&2; exit 1; }
done
for status in 200 201 400; do
  grep -Eq "^[[:space:]]+['\"]${status}['\"]:" "${spec}" || { echo "missing status ${status}" >&2; exit 1; }
done
go test ./tests/contracts -count=1
