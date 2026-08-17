#!/usr/bin/env bash
set -euo pipefail

project="emovis-component-dynamodb"
cleanup() {
  docker compose -p "${project}" -f compose.yaml --profile component-dynamodb down --remove-orphans --volumes >/dev/null
}
trap cleanup EXIT

cleanup
docker compose -p "${project}" -f compose.yaml --profile component-dynamodb up --detach dynamodb-local
for _ in $(seq 1 30); do
  if curl --silent --output /dev/null http://127.0.0.1:58000/; then break; fi
  sleep 1
done
curl --silent --output /dev/null http://127.0.0.1:58000/
mkdir -p .local/component-coverage
DYNAMODB_COMPONENT_ENDPOINT='http://127.0.0.1:58000' \
  go test ./tests/component -run TestDynamoDBLocalSatisfiesTransactionStoreContract -count=1 \
    -coverpkg=github.com/waggertron/emovis-transaction-intake/internal/transaction/adapters/dynamodb \
    -coverprofile=.local/component-coverage/dynamodb.out
go tool cover -func=.local/component-coverage/dynamodb.out >.local/component-coverage/dynamodb.txt

cleanup
trap - EXIT
if [[ -n "$(docker ps --filter "label=com.docker.compose.project=${project}" --quiet)" ]]; then
  echo "DynamoDB component containers remain after cleanup" >&2
  exit 1
fi
