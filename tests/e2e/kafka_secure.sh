#!/usr/bin/env bash
set -euo pipefail
source tests/e2e/lib.sh

e2e_setup kafka-secure
transaction_id="018f47a8-40d1-7e32-b6d6-4f4f8f9c9e28"
e2e_compose --profile e2e-kafka-secure up --build --detach \
  kafka-secure-init kafka-secure topic-bootstrap-secure e2e-kafka-secure
e2e_wait_for_api 18084
e2e_request 18084 "${transaction_id}"
e2e_compose exec -T kafka-secure /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server kafka-secure:9094 --consumer.config /etc/kafka/secrets/client.properties \
  --topic transaction.review-candidates.v1 --from-beginning --max-messages 1 --timeout-ms 30000 \
  >"${E2E_TEMP_DIR}/event.json"
grep -Fq '"eventType":"transaction.review-candidate"' "${E2E_TEMP_DIR}/event.json"
grep -Fq "\"source_reference\":\"${transaction_id}\"" "${E2E_TEMP_DIR}/event.json"
e2e_assert_replay_and_conflict 18084
