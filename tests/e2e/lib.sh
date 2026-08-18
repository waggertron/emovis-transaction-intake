#!/usr/bin/env bash
set -euo pipefail

e2e_setup() {
  local name="$1"
  E2E_PROJECT="emovis-e2e-${name}-$$"
  E2E_TEMP_DIR="$(mktemp -d -t "emovis-${name}-e2e.XXXXXX")"
  export E2E_PROJECT E2E_TEMP_DIR
  trap e2e_cleanup EXIT
}

e2e_compose() {
  docker compose -p "${E2E_PROJECT}" -f compose.yaml "$@"
}

e2e_compose_up_with_retry() {
  local attempt=1
  local max_attempts="${E2E_COMPOSE_MAX_ATTEMPTS:-3}"
  local retry_delay_seconds="${E2E_COMPOSE_RETRY_DELAY_SECONDS:-5}"

  until e2e_compose "$@"; do
    if (( attempt >= max_attempts )); then
      echo "Compose startup failed after ${max_attempts} attempts" >&2
      return 1
    fi

    echo "Compose startup failed (attempt ${attempt}/${max_attempts}); retrying in ${retry_delay_seconds}s" >&2
    sleep "${retry_delay_seconds}"
    ((attempt += 1))
  done
}

e2e_cleanup() {
  e2e_compose --profile "*" down --remove-orphans --volumes >/dev/null 2>&1 || true
  if [[ -d "${E2E_TEMP_DIR:-}" ]]; then
    find "${E2E_TEMP_DIR}" -type f -delete
    rmdir "${E2E_TEMP_DIR}"
  fi
  if [[ -n "$(docker ps --all --filter "label=com.docker.compose.project=${E2E_PROJECT}" --quiet)" ]]; then
    echo "Compose containers remain for ${E2E_PROJECT}" >&2
    return 1
  fi
  if [[ -n "$(docker volume ls --filter "label=com.docker.compose.project=${E2E_PROJECT}" --quiet)" ]]; then
    echo "Compose volumes remain for ${E2E_PROJECT}" >&2
    return 1
  fi
  if [[ -n "$(docker network ls --filter "label=com.docker.compose.project=${E2E_PROJECT}" --quiet)" ]]; then
    echo "Compose networks remain for ${E2E_PROJECT}" >&2
    return 1
  fi
}

e2e_wait_for_api() {
  local port="$1"
  curl --fail --silent --show-error --retry 40 --retry-delay 1 --retry-all-errors \
    "http://127.0.0.1:${port}/healthz" >"${E2E_TEMP_DIR}/health.json"
  grep -Fq '"status":"ok"' "${E2E_TEMP_DIR}/health.json"
}

e2e_request() {
  local port="$1"
  local transaction_id="$2"
  E2E_REQUEST="{\"source\":\"e2e-source\",\"source_reference\":\"${transaction_id}\",\"transaction_type\":\"toll\",\"transaction_time_utc\":\"2026-08-17T18:30:00Z\",\"base_amount\":\"7.25\",\"currency\":\"USD\",\"transponder_number\":\"0180012345678\",\"location\":{\"lane\":9007199254740993},\"metadata\":{\"rate\":12.50}}"
  export E2E_REQUEST
  local status
  status="$(curl --silent --show-error -D "${E2E_TEMP_DIR}/first.headers" -o "${E2E_TEMP_DIR}/first.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
    --data "${E2E_REQUEST}" "http://127.0.0.1:${port}/ingest/v1/transactions")"
  [[ "${status}" == "201" ]] || { echo "expected first request 201, got ${status}" >&2; return 1; }
}

e2e_assert_replay_and_conflict() {
  local port="$1"
  local status
  status="$(curl --silent --show-error -D "${E2E_TEMP_DIR}/replay.headers" -o "${E2E_TEMP_DIR}/replay.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
    --data "${E2E_REQUEST}" "http://127.0.0.1:${port}/ingest/v1/transactions")"
	[[ "${status}" == "200" ]] || { echo "expected replay 200, got ${status}" >&2; return 1; }
	grep -Eiq '^Idempotent-Replay: true' "${E2E_TEMP_DIR}/replay.headers"
	local initial_id replay_id
	initial_id="$(sed -nE 's/.*"id":"([^"]+)".*/\1/p' "${E2E_TEMP_DIR}/first.json")"
	replay_id="$(sed -nE 's/.*"id":"([^"]+)".*/\1/p' "${E2E_TEMP_DIR}/replay.json")"
	[[ -n "${initial_id}" && "${replay_id}" == "${initial_id}" ]] || { echo "expected replay to return original transaction ID, got ${replay_id} after ${initial_id}" >&2; return 1; }

  local changed="${E2E_REQUEST/\"base_amount\":\"7.25\"/\"base_amount\":\"7.26\"}"
  status="$(curl --silent --show-error -o "${E2E_TEMP_DIR}/conflict.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
    --data "${changed}" "http://127.0.0.1:${port}/ingest/v1/transactions")"
  [[ "${status}" == "400" ]] || { echo "expected changed duplicate 400, got ${status}" >&2; return 1; }
}

e2e_consume_event() {
  local transaction_id="$1"
  e2e_compose exec -T kafka /opt/kafka/bin/kafka-console-consumer.sh \
    --bootstrap-server kafka:9092 --topic transaction.review-candidates.v1 \
    --from-beginning --max-messages 1 --timeout-ms 30000 >"${E2E_TEMP_DIR}/event.json"
  grep -Fq '"eventType":"transaction.review-candidate"' "${E2E_TEMP_DIR}/event.json"
  grep -Fq "\"source_reference\":\"${transaction_id}\"" "${E2E_TEMP_DIR}/event.json"
  grep -Fq '"lane":9007199254740993' "${E2E_TEMP_DIR}/event.json"
  grep -Fq '"rate":12.50' "${E2E_TEMP_DIR}/event.json"
}

e2e_assert_dependency_failure() {
  local port="$1"
  local transaction_id="$2"
  local request="{\"source\":\"e2e-source\",\"source_reference\":\"${transaction_id}\",\"transaction_type\":\"toll\",\"transaction_time_utc\":\"2026-08-17T18:31:00Z\",\"base_amount\":\"8.25\",\"currency\":\"USD\",\"transponder_number\":\"0180012345678\"}"
  local status
  status="$(curl --silent --show-error -o "${E2E_TEMP_DIR}/dependency-failure.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -H 'X-API-Key: local-development-only-key' \
    --data "${request}" "http://127.0.0.1:${port}/ingest/v1/transactions")"
  [[ "${status}" == "503" ]] || { echo "expected dependency failure 503, got ${status}" >&2; return 1; }
}
