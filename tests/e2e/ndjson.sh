#!/usr/bin/env bash
set -euo pipefail
source tests/e2e/lib.sh

e2e_setup ndjson
transaction_id="018f47a8-40d1-7e32-b6d6-4f4f8f9c9e24"
e2e_compose --profile e2e-ndjson up --build --detach kafka topic-bootstrap e2e-ndjson-init e2e-ndjson
e2e_wait_for_api 18080
e2e_request 18080 "${transaction_id}"
e2e_consume_event "${transaction_id}"
e2e_compose stop e2e-ndjson
e2e_compose --profile e2e-ndjson up --detach e2e-ndjson
e2e_wait_for_api 18080
e2e_assert_replay_and_conflict 18080
