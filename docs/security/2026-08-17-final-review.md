# Final security review — 2026-08-17

## Result

Release assessment: **pass after remediation**. No unresolved Critical, High, or Medium finding remains in the locally reviewable product, container, or reference infrastructure. Real AWS ingress, organization policy, runtime secret population, and destructive lifecycle controls still require an authorized platform review before deployment; this repository does not apply cloud resources.

## Scope and method

The review covered the Go HTTP/API and process lifecycle, API-key authentication, input limits and validation, request/error logging, storage adapters and SQL parameterization, NDJSON permissions, outbox concurrency, Kafka TLS/SCRAM and payload boundaries, local/AWS secret providers, dependencies, race behavior, Docker build/runtime, Compose exposure and cleanup, CI permissions, Kubernetes workload security, Terraform network/IAM/encryption/deletion controls, and source plus Git-history secrets.

Canonical evidence:

- `make test-race`, `make lint`, OpenAPI/container/CI contracts: pass.
- `make security-vuln`: pass after remediation; `govulncheck` 1.1.4 reports no reachable vulnerabilities with Go 1.26.6.
- `make security-secrets`: pass; Gitleaks 8.24.2 finds no working-tree or 12-commit-history leaks with only exact sentinel-key matches allowed.
- `make security-config`: pass for High/Critical source, dependency, secret, Dockerfile, Kubernetes, and Terraform findings.
- `make security-image`: pass; Trivy 0.59.1 reports 0 High and 0 Critical findings in the distroless production image.
- A separate Medium+ IaC scan reports no remaining finding after remediation.
- `make terraform-validate`, `make terraform-plan`, `make k8s-validate`, and `make test-infrastructure`: pass without credentials, backend, cluster, or apply. The plan contains 61 creates, 0 changes, and 0 destroys.

Trivy 0.59.1's downloadable-policy update emitted a Rego compatibility error for one unrelated EC2 AMI-owner rule. The repeatable gate therefore uses that version's embedded checks (`--skip-check-update`) and supplements them with repository policy assertions. This limitation is not represented as scanner coverage.

## Findings

### SEC-001 — Reachable vulnerable Go runtime and text dependency

- Severity/status: **High — resolved**.
- Location: `go.mod:3-5`, `Dockerfile:1`.
- Evidence: the initial reachable scan found seven standard-library advisories under Go 1.26.4 plus GO-2026-5970 in `golang.org/x/text` 0.29.0.
- Impact: crafted network, certificate, URL, XML, or text inputs could trigger denial of service or protocol weakness in reachable paths.
- Fix: pin toolchain/container builder to Go 1.26.6 and upgrade `golang.org/x/text` to 0.39.0 (with the compatible `x/sync` update).
- Follow-up: full Go tests pass under 1.26.6 and the reachable scan reports no vulnerabilities.

### SEC-002 — RDS IAM authentication disabled

- Severity/status: **Medium — resolved**.
- Location: `infra/terraform/main.tf:389`.
- Evidence: AVD-AWS-0176 reported the PostgreSQL instance without IAM database authentication.
- Impact: operators would be limited to long-lived database credentials even where short-lived AWS identity is appropriate.
- Fix: set `iam_database_authentication_enabled = true`; Secrets Manager-managed credentials remain available for the selected application integration.
- Follow-up: Terraform validation/plan and the Medium+ IaC rescan pass.

### SEC-003 — VPC traffic lacked flow-log evidence

- Severity/status: **Medium — resolved**.
- Location: `infra/terraform/main.tf:13-56`.
- Evidence: AVD-AWS-0178 reported no VPC Flow Logs.
- Impact: incident response and anomalous network-traffic investigation would lack a basic audit source.
- Fix: add all-traffic flow logs with a dedicated service role, least-purpose log permissions, KMS-encrypted CloudWatch group, 30-day retention, and 60-second aggregation.
- Follow-up: infrastructure policy tests, Terraform validation/plan, and the Medium+ IaC rescan pass.

### SEC-004 — Destructive RDS default

- Severity/status: **Medium — resolved; local-plan exception is non-applying**.
- Location: `infra/terraform/variables.tf` (`deletion_protection`), `infra/terraform/terraform.tfvars.example`.
- Evidence: AVD-AWS-0177 reported the earlier default-off deletion protection.
- Impact: an applied environment could permit accidental database deletion without a final snapshot.
- Fix: default deletion protection on; this also makes final snapshots mandatory. The credential-free disposable example explicitly sets it off only to demonstrate both conditional paths in a plan that cannot apply and is documented as unsuitable for deployment.
- Follow-up: repository policy asserts the safe default and the Medium+ IaC rescan reports no finding.

### SEC-005 — Terraform provider cache entered Docker build context

- Severity/status: **Medium — resolved**.
- Location: `.dockerignore`, `tests/containers/definitions_test.sh`.
- Evidence: the first production build transferred 832.82 MB because nested `.terraform` provider binaries were included in context.
- Impact: unnecessary third-party binaries and local artifacts crossed the build boundary, increasing leakage and supply-chain risk.
- Fix: exclude nested `.terraform`, plans, and state artifacts and enforce those entries through the container contract.
- Follow-up: the rebuilt context is 13.76 kB and the resulting image has 0 High/Critical findings.

### SEC-006 — Local sentinel API keys matched secret heuristics

- Severity/status: **Informational — false positive, narrowly suppressed**.
- Location: `.gitleaks.toml`, README/local smoke/E2E requests.
- Evidence: ten Gitleaks matches were exclusively `local-development-only-key` or `rotated-local-key`; history contained no other candidate.
- Impact: untreated false positives would either fail every gate or encourage unsafe path-wide exclusions.
- Fix: allow only an `X-API-Key` header containing those two exact sentinel values; do not exclude files or generic API-key patterns.
- Follow-up: working-tree and full-history scans pass.

## Control review

- HTTP: 64 KiB strict JSON body, unknown/trailing value rejection, bounded headers/timeouts, validated request IDs, generic dependency errors, and bounded graceful shutdown.
- Authentication: API keys are SHA-256 digested and constant-time compared; partner identity derives from the key rather than request data. Production rotation requires a controlled restart, which is documented and exercised locally.
- Data: domain validation prevents invalid writes; SQL is parameterized; each store atomically accepts with outbox intent; lease/retry races are tested; NDJSON is owner-only, append/fsync based, and restricted to one process.
- Kafka: TLS 1.2 minimum, certificate/hostname verification, SCRAM-SHA-512, required-all acknowledgements, stable keys/event IDs, no credential fields, and explicit consumer deduplication responsibility.
- Secrets: bounded JSON, no value-bearing provider errors, mutually exclusive provider selection, environment precedence, IRSA-scoped AWS lookup, no plaintext Kubernetes/Terraform values, and exact-resource IAM.
- Containers/Kubernetes: distroless non-root runtime, immutable deployment digests, read-only root filesystems, dropped capabilities, seccomp, bounded resources, probes, HPAs, and disruption budgets. The Service is ClusterIP; ingress/TLS/WAF decisions are intentionally outside this non-applying reference.
- AWS: private EKS endpoint/subnets, TLS-only MSK with SCRAM and KMS, encrypted DynamoDB/RDS/Secrets/observability, multi-AZ RDS, flow logs, broker logs/alarm, service-account-specific IRSA trust, and safe deletion defaults.
- CI: read-only repository permission, no cloud credentials or apply, canonical Make gates, full Git fetch for history scanning, and separate bounded static/component/E2E/infrastructure/security jobs.

## Coverage and residual limits

Every production Go package independently exceeds 85% statement coverage: topic bootstrap 86.6%, transaction service 87.0%, bootstrap 93.6%, secrets 86.5%, DynamoDB 90.9%, HTTP 96.2%, Kafka 86.0%, memory 85.5%, NDJSON 92.5%, PostgreSQL 86.5%, application 89.1%, and domain 95.5%.

Component confidence is behavioral rather than represented by a misleading aggregate statement percentage: the production PostgreSQL (75.7% component-instrumented) and DynamoDB (74.7%) adapters run the shared store/outbox contract against real local services; plaintext and TLS/SCRAM Kafka publish/consume paths run against real brokers; local secrets are 86.5% component-instrumented and cover lookup/rotation/failure/redaction; and every local implementation has an HTTP→outbox→Kafka E2E target. Generated logs/profiles live only under self-cleaning `.local/` paths.

Residual deployment work is intentionally procedural: authorize and populate real secrets, choose/store remote Terraform state, replace image placeholders, approve ingress and certificate policy, execute the topic Job after MSK readiness, and run the separately documented non-production AWS smoke. None of those expands the locally validated security claim.
