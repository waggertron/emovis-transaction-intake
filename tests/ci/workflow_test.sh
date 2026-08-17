#!/usr/bin/env bash
set -euo pipefail

workflow=".github/workflows/ci.yml"
[[ -f "${workflow}" ]] || { echo "missing ${workflow}" >&2; exit 1; }
grep -Fq 'contents: read' "${workflow}"
grep -Fq 'actions/checkout@v7' "${workflow}"
grep -Fq 'actions/setup-go@v7' "${workflow}"
for target in test test-race coverage lint build compose-config smoke test-component test-e2e terraform-fmt terraform-validate k8s-validate test-infrastructure; do
  grep -Fq "make ${target}" "${workflow}" || { echo "CI missing make ${target}" >&2; exit 1; }
done
for target in security-vuln security-secrets security-config security-image; do
  grep -Fq "make ${target}" "${workflow}" || { echo "CI missing make ${target}" >&2; exit 1; }
done
if grep -Eiq 'aws[_ -]?(access|secret)|terraform apply|kubectl apply' "${workflow}"; then
  echo "CI must not contain cloud credentials or apply infrastructure" >&2
  exit 1
fi
