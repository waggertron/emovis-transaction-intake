# Adversarial review follow-up — 2026-08-17

## Result

The prior release-pass assertion was reopened after a whole-repository adversarial review identified three High, four Medium, and one Low finding outside the effective reach of the existing scanners. All findings are remediated. The complete local gate passed on 2026-08-17; final command evidence is recorded in the archived remediation plan.

## Findings and remediation

### ADV-001 — Stale outbox completion

- Severity/status: **High — remediated and validated**.
- Evidence: completion previously used event ID only. Claims now carry opaque tokens and memory, NDJSON, PostgreSQL, and DynamoDB reject stale success/failure transitions with `ErrLeaseLost`.
- Validation: focused stale-owner unit test plus the shared real-store component contract.

### ADV-002 — DynamoDB filtered-page starvation

- Severity/status: **High — remediated and validated**.
- Evidence: the adapter previously stopped after one filtered `Query`. It now follows `LastEvaluatedKey` until the batch is full or the candidate set is exhausted.
- Validation: deterministic multi-page unit test and DynamoDB Local component contract.

### ADV-003 — Incomplete topic-bootstrap deployment

- Severity/status: **High — remediated and validated**.
- Evidence: Kubernetes Secret references were removed. Topic bootstrap loads bounded configuration directly from local files or AWS Secrets Manager through IRSA. Terraform gates SCRAM association and the Job on `runtime_secrets_ready` so secret population has an explicit two-stage order.
- Validation: command provider tests, secure-Kafka component test, parsed manifests, infrastructure contracts, and Terraform validation.

### ADV-004 — Concurrent intake race classification

- Severity/status: **Medium — remediated and validated**.
- Evidence: DynamoDB rereads identity after conditional cancellation; PostgreSQL classifies uniqueness/serialization SQL states after rollback.
- Validation: unit classification tests and simultaneous identical acceptance against PostgreSQL and DynamoDB Local.

### ADV-005 — Constant readiness

- Severity/status: **Medium — remediated and validated**.
- Evidence: production readiness now performs a one-second bounded check against the selected store. Kafka remains outside API readiness because publication is asynchronous.

### ADV-006 — EKS/image architecture mismatch

- Severity/status: **Medium — remediated and validated**.
- Evidence: `make build-arm64` produces and inspects Linux ARM64 commands, while `make image-arm64` builds and inspects the production ARM64 image for the selected Graviton nodes; CI executes both targets.

### ADV-007 — Presence-only validation

- Severity/status: **Medium — remediated and validated**.
- Evidence: OpenAPI is parsed into a structured model; rendered Kubernetes validation checks namespace, service-account and Secret references, selectors, images, resources, and required runtime environment.

### ADV-008 — HTTP/OpenAPI edge mismatch

- Severity/status: **Low — remediated and validated**.
- Evidence: transaction intake requires `application/json`, returns `415` otherwise, and the contract documents `405` and `415` behavior.

## Residual boundary

The checked-in AWS topology remains non-applying. Real secret values, image digests, ingress/TLS/WAF choices, remote Terraform state, and an authorized non-production apply remain operator-controlled deployment inputs and are not represented as locally proven cloud execution.
