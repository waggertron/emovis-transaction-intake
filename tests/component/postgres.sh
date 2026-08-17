#!/usr/bin/env bash
set -euo pipefail

project="emovis-component-postgres"
cleanup() {
  docker compose -p "${project}" -f compose.yaml --profile component-postgres down --remove-orphans --volumes >/dev/null
}
trap cleanup EXIT

cleanup
docker compose -p "${project}" -f compose.yaml --profile component-postgres up --detach --wait postgres
mkdir -p .local/component-coverage
POSTGRES_COMPONENT_URL='postgres://transaction_test:local-component-only@127.0.0.1:55432/transactions?sslmode=disable' \
  go test ./tests/component -run TestPostgresSatisfiesTransactionStoreContract -count=1 \
    -coverpkg=github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/postgres \
    -coverprofile=.local/component-coverage/postgres.out
go tool cover -func=.local/component-coverage/postgres.out >.local/component-coverage/postgres.txt

cleanup
trap - EXIT
if [[ -n "$(docker ps --filter "label=com.docker.compose.project=${project}" --quiet)" ]]; then
  echo "PostgreSQL component containers remain after cleanup" >&2
  exit 1
fi
