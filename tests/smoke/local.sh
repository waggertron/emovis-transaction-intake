#!/usr/bin/env bash
set -euo pipefail

project="emovis-transaction-intake"
temp_dir="$(mktemp -d -t emovis-smoke.XXXXXX)"

cleanup() {
  docker compose -p "${project}" -f compose.yaml down --remove-orphans --volumes >/dev/null
  find "${temp_dir}" -type f -delete
  rmdir "${temp_dir}"
}
trap cleanup EXIT

make --no-print-directory compose-up
curl --fail --silent --show-error --retry 20 --retry-delay 1 --retry-all-errors \
  http://127.0.0.1:8080/healthz >"${temp_dir}/health.json"
grep -Fq '"status":"ok"' "${temp_dir}/health.json"

request='{"transactionId":"018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01","occurredAt":"2026-08-16T20:30:00Z","amountMinor":725,"currency":"USD","agencyId":"agency-17","plazaId":"plaza-4","laneId":"lane-2","vehicleClass":"CAR"}'
status="$(curl --silent --show-error -D "${temp_dir}/first.headers" -o "${temp_dir}/first.json" -w '%{http_code}' \
  -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
  --data "${request}" http://127.0.0.1:8080/v1/transactions)"
[[ "${status}" == "201" ]] || { echo "expected first request 201, got ${status}" >&2; exit 1; }

status="$(curl --silent --show-error -D "${temp_dir}/replay.headers" -o "${temp_dir}/replay.json" -w '%{http_code}' \
  -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
  --data "${request}" http://127.0.0.1:8080/v1/transactions)"
[[ "${status}" == "200" ]] || { echo "expected replay 200, got ${status}" >&2; exit 1; }
grep -Eiq '^Idempotent-Replay: true' "${temp_dir}/replay.headers"

changed="${request/\"amountMinor\":725/\"amountMinor\":726}"
status="$(curl --silent --show-error -o "${temp_dir}/conflict.json" -w '%{http_code}' \
  -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
  --data "${changed}" http://127.0.0.1:8080/v1/transactions)"
[[ "${status}" == "409" ]] || { echo "expected changed duplicate 409, got ${status}" >&2; exit 1; }

set +e
docker compose -p "${project}" -f compose.yaml exec -T kafka \
  /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 \
  --topic transaction.review-candidates.v1 --from-beginning --max-messages 2 --timeout-ms 10000 \
  --property print.key=true --property 'key.separator=|' \
  >"${temp_dir}/event.json"
set -e
[[ "$(wc -l <"${temp_dir}/event.json" | tr -d ' ')" == "1" ]] || { echo "expected exactly one Kafka event" >&2; exit 1; }
grep -Fq '"eventType":"transaction.review-candidate"' "${temp_dir}/event.json"
grep -Fq '"schemaVersion":1' "${temp_dir}/event.json"
grep -Eq '"eventId":"[^"]+"' "${temp_dir}/event.json"
grep -Fq '"transactionId":"018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01"' "${temp_dir}/event.json"
grep -Fq 'local-partner:018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01|' "${temp_dir}/event.json"
if grep -Eiq 'api.?key|authorization|sasl|password|payment' "${temp_dir}/event.json"; then
  echo "Kafka event contains a forbidden credential or payment field" >&2
  exit 1
fi

cleanup
trap - EXIT
if [[ -n "$(docker ps --filter "label=com.docker.compose.project=${project}" --quiet)" ]]; then
  echo "Compose project containers remain after cleanup" >&2
  exit 1
fi
