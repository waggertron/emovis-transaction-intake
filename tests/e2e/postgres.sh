#!/usr/bin/env bash
set -euo pipefail
source tests/e2e/lib.sh

e2e_setup postgres
transaction_id="018f47a8-40d1-7e32-b6d6-4f4f8f9c9e25"
e2e_compose --profile e2e-postgres up --build --detach postgres kafka topic-bootstrap e2e-postgres-api e2e-postgres-worker
e2e_wait_for_api 18081
e2e_request 18081 "${transaction_id}"
e2e_consume_event "${transaction_id}"
e2e_compose restart e2e-postgres-api e2e-postgres-worker
e2e_wait_for_api 18081
e2e_assert_replay_and_conflict 18081
e2e_compose stop postgres
e2e_assert_dependency_failure 18081 "018f47a8-40d1-7e32-b6d6-4f4f8f9c9f25"
