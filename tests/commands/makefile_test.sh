#!/usr/bin/env bash
set -euo pipefail

required_targets=(
  help test test-unit lint format-check vet build
  run-api run-worker run-local compose-up compose-down
  compose-config smoke validate clean
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
