#!/usr/bin/env bash
set -euo pipefail

checker="tests/coverage/check.sh"
temp_dir="$(mktemp -d -t emovis-coverage-gate.XXXXXX)"
cleanup() {
  find "${temp_dir}" -type f -delete
  rmdir "${temp_dir}"
}
trap cleanup EXIT

cat >"${temp_dir}/packages.txt" <<'REPORT'
ok  example.test/cmd/service  0.1s  coverage: 85.0% of statements
ok  example.test/internal/domain  0.1s  coverage: 95.0% of statements
REPORT
cat >"${temp_dir}/expected.txt" <<'PACKAGES'
example.test/cmd/service
example.test/internal/domain
PACKAGES

bash "${checker}" "${temp_dir}/packages.txt" "${temp_dir}/expected.txt" 85

expect_failure() {
  local name="$1"
  if bash "${checker}" "${temp_dir}/${name}.txt" "${temp_dir}/expected.txt" 85 >/dev/null 2>&1; then
    echo "coverage gate unexpectedly accepted ${name}" >&2
    exit 1
  fi
}

cat >"${temp_dir}/below.txt" <<'REPORT'
ok  example.test/cmd/service  0.1s  coverage: 84.9% of statements
ok  example.test/internal/domain  0.1s  coverage: 100.0% of statements
REPORT
expect_failure below

cat >"${temp_dir}/false-aggregate.txt" <<'REPORT'
ok  example.test/cmd/service  0.1s  coverage: 20.0% of statements
ok  example.test/internal/domain  0.1s  coverage: 100.0% of statements
total: (statements) 95.0%
REPORT
expect_failure false-aggregate

cat >"${temp_dir}/missing.txt" <<'REPORT'
ok  example.test/internal/domain  0.1s  coverage: 95.0% of statements
REPORT
expect_failure missing

cat >"${temp_dir}/malformed.txt" <<'REPORT'
ok  example.test/cmd/service  0.1s  coverage unavailable
ok  example.test/internal/domain  0.1s  coverage: 95.0% of statements
REPORT
expect_failure malformed

: >"${temp_dir}/stale.txt"
touch -t 202001010000 "${temp_dir}/stale.txt"
expect_failure stale
