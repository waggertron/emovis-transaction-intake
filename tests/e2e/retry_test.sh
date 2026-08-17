#!/usr/bin/env bash
set -euo pipefail
source tests/e2e/lib.sh

calls=0
e2e_compose() {
  ((calls += 1))
  [[ "${calls}" -ge 3 ]]
}
sleep() { :; }

E2E_COMPOSE_MAX_ATTEMPTS=3
E2E_COMPOSE_RETRY_DELAY_SECONDS=0
e2e_compose_up_with_retry --profile e2e-dynamodb up --build --detach dynamodb-local
[[ "${calls}" == "3" ]] || { echo "expected three Compose attempts, got ${calls}" >&2; exit 1; }

calls=0
e2e_compose() {
  ((calls += 1))
  return 1
}

if e2e_compose_up_with_retry --profile e2e-dynamodb up --build --detach dynamodb-local; then
  echo "expected Compose startup retries to fail after their limit" >&2
  exit 1
fi
[[ "${calls}" == "3" ]] || { echo "expected three failed Compose attempts, got ${calls}" >&2; exit 1; }
