#!/usr/bin/env bash
set -euo pipefail

# The established smoke test is the production combined-local memory E2E path.
bash tests/smoke/local.sh
