#!/usr/bin/env bash
set -euo pipefail

spec="api/openapi.yaml"
[[ -f "${spec}" ]] || { echo "missing ${spec}" >&2; exit 1; }
grep -Fq 'openapi: 3.1.0' "${spec}"
grep -Fq 'Mocked contract' "${spec}"
for path in /v1/transactions /healthz /readyz /metrics; do
  grep -Fq "${path}:" "${spec}" || { echo "missing path ${path}" >&2; exit 1; }
done
for field in transactionId occurredAt amountMinor currency agencyId plazaId laneId vehicleClass; do
  grep -Fq "${field}:" "${spec}" || { echo "missing field ${field}" >&2; exit 1; }
done
for status in 200 201 400 401 409 413 422 503; do
  grep -Eq "^[[:space:]]+'${status}':" "${spec}" || { echo "missing status ${status}" >&2; exit 1; }
done
grep -Fq 'Idempotent-Replay:' "${spec}"
grep -Fq 'X-API-Key' "${spec}"
go test ./tests/contracts -run TestOpenAPIContractIsStructurallyComplete -count=1
