#!/usr/bin/env bash
set -euo pipefail

mkdir -p .local/component-coverage
go test ./internal/bootstrap/secrets -count=1 -coverprofile=.local/component-coverage/secrets.out
go test ./cmd/transaction-service -run 'TestRunLoadsConfigurationFromLocalSecretFile|TestRunDoesNotStartWithInvalidConfig' -count=1
go tool cover -func=.local/component-coverage/secrets.out >.local/component-coverage/secrets.txt
