#!/usr/bin/env bash
set -euo pipefail

project="emovis-component-kafka-secure-$$"
cleanup() {
  docker compose -p "${project}" -f compose.yaml --profile "*" down --remove-orphans --volumes >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -p "${project}" -f compose.yaml --profile component-kafka-secure up --build --detach \
  kafka-secure-init kafka-secure topic-bootstrap-secure
docker compose -p "${project}" -f compose.yaml --profile component-kafka-secure wait topic-bootstrap-secure
docker compose -p "${project}" -f compose.yaml exec -T kafka-secure \
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka-secure:9094 \
  --command-config /etc/kafka/secrets/client.properties --list | grep -Fxq transaction.review-candidates.v1
if docker compose -p "${project}" -f compose.yaml exec -T kafka-secure \
  /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka-secure:9094 \
  --command-config /etc/kafka/secrets/bad-client.properties --list >/dev/null 2>&1; then
  echo "secure Kafka accepted incorrect SCRAM credentials" >&2
  exit 1
fi
if docker compose -p "${project}" -f compose.yaml --profile component-kafka-secure run --rm \
  -e KAFKA_SASL_USERNAME= -e KAFKA_SASL_PASSWORD= topic-bootstrap-secure >/dev/null 2>&1; then
  echo "secure Kafka accepted absent SCRAM credentials" >&2
  exit 1
fi
if docker compose -p "${project}" -f compose.yaml --profile component-kafka-secure run --rm \
  -e KAFKA_CA_FILE= topic-bootstrap-secure >/dev/null 2>&1; then
  echo "secure Kafka accepted an untrusted ephemeral certificate" >&2
  exit 1
fi

cleanup
trap - EXIT
mkdir -p .local/component-coverage
printf '%s\n' 'PASS: production TLS/SCRAM bootstrap and client, invalid credentials, certificate verification, and cleanup' >.local/component-coverage/kafka-secure.txt
[[ -z "$(docker ps --all --filter "label=com.docker.compose.project=${project}" --quiet)" ]]
[[ -z "$(docker volume ls --filter "label=com.docker.compose.project=${project}" --quiet)" ]]
[[ -z "$(docker network ls --filter "label=com.docker.compose.project=${project}" --quiet)" ]]
