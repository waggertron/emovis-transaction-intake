#!/usr/bin/env bash
set -euo pipefail

required_targets=(
  help test test-unit test-race test-contract lint format-check vet build
  run-api run-worker run-local compose-up compose-down
  compose-config smoke coverage validate-static validate clean
  test-component test-component-storage test-component-postgres
  test-component-dynamodb test-component-kafka test-component-kafka-secure
  test-component-secrets
  test-e2e test-e2e-memory test-e2e-ndjson test-e2e-postgres
  test-e2e-dynamodb test-e2e-secrets test-e2e-kafka-secure
  test-cloud-equivalence
  terraform-fmt terraform-init terraform-validate terraform-plan
  k8s-validate test-infrastructure docs-validate
  security security-vuln security-secrets security-config security-image
)

[[ -f Makefile ]] || { echo "missing Makefile" >&2; exit 1; }

help_output="$(make --no-print-directory help)"
for target in "${required_targets[@]}"; do
  grep -Eq "^${target}:" Makefile || { echo "missing Make target: ${target}" >&2; exit 1; }
  grep -Eq "(^|[[:space:]])${target}([[:space:]]|$)" <<<"${help_output}" || {
    echo "Make help does not document: ${target}" >&2
    exit 1
  }
done

validate_definition="$(grep -E '^validate:' Makefile)"
for dependency in validate-static test-component test-e2e terraform-validate k8s-validate test-infrastructure security; do
  grep -Eq "(^|[[:space:]])${dependency}([[:space:]]|$)" <<<"${validate_definition}" || {
    echo "validate does not require ${dependency}" >&2
    exit 1
  }
done

grep -Fq '.SHELLFLAGS := -eu -o pipefail -c' Makefile || {
  echo "Makefile must enable strict shell flags" >&2
  exit 1
}
for binary in transaction-service topic-bootstrap; do
  grep -Fq ".local/bin/${binary}" Makefile || {
    echo "build output is not isolated for: ${binary}" >&2
    exit 1
  }
done
