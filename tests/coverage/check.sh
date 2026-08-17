#!/usr/bin/env bash
set -euo pipefail

report="${1:?usage: check.sh REPORT EXPECTED_PACKAGES THRESHOLD}"
expected="${2:?usage: check.sh REPORT EXPECTED_PACKAGES THRESHOLD}"
threshold="${3:?usage: check.sh REPORT EXPECTED_PACKAGES THRESHOLD}"

[[ -s "${report}" ]] || { echo "coverage report is missing or empty: ${report}" >&2; exit 1; }
[[ -s "${expected}" ]] || { echo "expected package list is missing or empty: ${expected}" >&2; exit 1; }
if [[ -n "$(find "${report}" -mmin +60 -print)" ]]; then
  echo "coverage report is stale: ${report}" >&2
  exit 1
fi

failed=0
while IFS= read -r package; do
  [[ -n "${package}" ]] || continue
  coverage="$(awk -v wanted="${package}" '
    $1 == "ok" && $2 == wanted {
      for (field = 3; field <= NF; field++) {
        if ($field == "coverage:") {
          value = $(field + 1)
          sub(/%$/, "", value)
          print value
          exit
        }
      }
    }
  ' "${report}")"
  if [[ -z "${coverage}" ]]; then
    echo "missing or malformed coverage for ${package}" >&2
    failed=1
    continue
  fi
  if ! awk -v actual="${coverage}" -v minimum="${threshold}" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
    echo "${package}: ${coverage}% is below ${threshold}%" >&2
    failed=1
  else
    echo "${package}: ${coverage}%"
  fi
done <"${expected}"

exit "${failed}"
