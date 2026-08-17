#!/usr/bin/env bash
set -euo pipefail

# The memory smoke path uses production topic bootstrap, publisher, and broker
# consumption while also proving idempotent intake does not create a second event.
bash tests/smoke/local.sh
KAFKA_COMPONENT_UNAVAILABLE=1 go test ./tests/component -run TestKafkaPublisherReportsUnavailableBroker -count=1
mkdir -p .local/component-coverage
printf '%s\n' 'PASS: production topic bootstrap, publisher, broker consumption, and idempotent one-event path' >.local/component-coverage/kafka.txt
