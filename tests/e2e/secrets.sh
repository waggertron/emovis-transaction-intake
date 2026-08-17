#!/usr/bin/env bash
set -euo pipefail
source tests/e2e/lib.sh

e2e_setup secrets
secret_file="${E2E_TEMP_DIR}/config.json"
export LOCAL_SECRET_FILE_HOST="${secret_file}"
umask 077
printf '%s' '{"API_KEY":"local-development-only-key","PARTNER_ID":"local-partner","HTTP_ADDRESS":":8080","KAFKA_BROKERS":"kafka:9092","KAFKA_TOPIC":"transaction.review-candidates.v1","STORE_DRIVER":"ndjson","STORE_PATH":"/data/transactions.ndjson"}' >"${secret_file}"

transaction_id="018f47a8-40d1-7e32-b6d6-4f4f8f9c9e27"
e2e_compose --profile e2e-secrets up --build --detach kafka topic-bootstrap e2e-ndjson-init e2e-secrets
e2e_wait_for_api 18083
e2e_request 18083 "${transaction_id}"
e2e_consume_event "${transaction_id}"

printf '%s' '{"API_KEY":"rotated-local-key","PARTNER_ID":"local-partner","HTTP_ADDRESS":":8080","KAFKA_BROKERS":"kafka:9092","KAFKA_TOPIC":"transaction.review-candidates.v1","STORE_DRIVER":"ndjson","STORE_PATH":"/data/transactions.ndjson"}' >"${secret_file}"
e2e_compose restart e2e-secrets
e2e_wait_for_api 18083

status="$(curl --silent --show-error -o "${E2E_TEMP_DIR}/old-key.json" -w '%{http_code}' \
  -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
  --data "${E2E_REQUEST}" http://127.0.0.1:18083/v1/transactions)"
[[ "${status}" == "401" ]] || { echo "expected rotated old key rejection 401, got ${status}" >&2; exit 1; }
status="$(curl --silent --show-error -D "${E2E_TEMP_DIR}/rotated.headers" -o "${E2E_TEMP_DIR}/rotated.json" -w '%{http_code}' \
  -H 'Content-Type: application/json' -H 'X-API-Key: rotated-local-key' \
  --data "${E2E_REQUEST}" http://127.0.0.1:18083/v1/transactions)"
[[ "${status}" == "200" ]] || { echo "expected rotated-key replay 200, got ${status}" >&2; exit 1; }
grep -Eiq '^Idempotent-Replay: true' "${E2E_TEMP_DIR}/rotated.headers"

e2e_compose logs --no-color e2e-secrets >"${E2E_TEMP_DIR}/service.log"
if grep -Fq 'local-development-only-key' "${E2E_TEMP_DIR}/service.log" || grep -Fq 'rotated-local-key' "${E2E_TEMP_DIR}/service.log"; then
  echo "secret value leaked to service logs" >&2
  exit 1
fi
