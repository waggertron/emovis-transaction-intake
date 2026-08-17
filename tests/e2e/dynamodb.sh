#!/usr/bin/env bash
set -euo pipefail
source tests/e2e/lib.sh

e2e_setup dynamodb
transaction_id="018f47a8-40d1-7e32-b6d6-4f4f8f9c9e26"
e2e_compose_up_with_retry --profile e2e-dynamodb up --build --detach dynamodb-init dynamodb-local kafka topic-bootstrap e2e-dynamodb-api e2e-dynamodb-worker
e2e_wait_for_api 18082
e2e_request 18082 "${transaction_id}"
e2e_consume_event "${transaction_id}"
e2e_compose restart e2e-dynamodb-api e2e-dynamodb-worker
e2e_wait_for_api 18082
e2e_assert_replay_and_conflict 18082
e2e_compose stop dynamodb-local
e2e_assert_dependency_failure 18082 "018f47a8-40d1-7e32-b6d6-4f4f8f9c9f26"
