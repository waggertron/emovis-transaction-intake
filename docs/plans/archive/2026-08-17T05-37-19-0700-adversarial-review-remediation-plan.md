# Adversarial review remediation plan

## Decision Notes

- **2026-08-17 — User — Remediate the whole-repository adversarial review through a general review plan.** The findings span correctness, concurrency, contracts, deployment, portability, testing, and security, so they must not be mislabeled as exclusively security work. Complete every item with local evidence and archive this plan last.
- **2026-08-17 — Joint — Preserve at-least-once Kafka delivery with fenced outbox ownership.** Claims will carry opaque ownership tokens and every success/failure transition will be conditional on the current token. Duplicate Kafka delivery remains possible, consumers still deduplicate immutable event IDs, and stale workers must never mutate reassigned records.
- **2026-08-17 — Codex — Make DynamoDB dispatch selection progress across filtered pages.** The adapter will continue through `LastEvaluatedKey` until it fills the requested eligible batch or exhausts candidates, preventing ineligible leading records from starving later work.
- **2026-08-17 — Codex — Classify concurrent intake races at the storage boundary.** DynamoDB conditional conflicts and PostgreSQL unique/serialization races will be reread or retried sufficiently to return replay/conflict rather than leaking expected concurrency as service unavailability.
- **2026-08-17 — Codex — Make readiness dependency-aware without coupling intake to Kafka availability.** API readiness will verify the selected intake store with a bounded check; Kafka remains asynchronous and therefore is not an API readiness dependency.
- **2026-08-17 — Codex — Use one IRSA-backed secret-loading path for application and topic bootstrap.** The bootstrap command will load its Kafka settings from AWS Secrets Manager directly, removing its undeclared Kubernetes Secret dependency while retaining externally populated secrets.
- **2026-08-17 — Codex — Align image validation with Graviton deployment.** The documented build/CI contract will explicitly produce or verify Linux ARM64 compatibility for the selected `m7g` EKS nodes.
- **2026-08-17 — Codex — Replace presence-only checks with semantic validation where practical.** OpenAPI and rendered deployment artifacts will be parsed and cross-checked for references and required runtime configuration, while repository-specific tests continue to cover behavior that generic schemas cannot prove.
- **2026-08-17 — User — Make README onboarding command-first and answer operational questions directly.** The README must explain how to start each useful mode with Make, where the local API key comes from, how to exercise the API, how to stop and clean the stack, and how to diagnose common failures. Any discovered command ergonomics issue must be reported and corrected where safe.
- **2026-08-17 — User — Dotenv templates must drive direct Go workflows before local fallbacks.** The canonical direct-run Make targets source a trusted `ENV_FILE` first and export its values to Go, while safe `LOCAL_*` values remain fallbacks. All non-example `.env*` files stay ignored, and the README must warn that sourcing is appropriate only for trusted local files.
- **2026-08-17 — Codex — Smoke runs must begin from an isolated named-project state.** An interrupted aggregate run demonstrated that cleanup only on exit permits a later run to reuse accepted in-memory state. The smoke harness will clean its repository-owned Compose project both before startup and on exit.

## Completion checklist

- [x] **1.1** Preserve the pre-existing user change in `docs/retrospective/2026-08-17-repository-initiation.md` and keep remediation changes isolated.
- [x] **1.2** Add this timestamped active plan, record every material decision above, and validate strict numbering and documentation links.
- [x] **2.1** Write focused failing application/store contract tests proving a stale expired claimant cannot mark a reassigned event published or failed.
- [x] **2.2** Add an opaque claim token to the outbox port and implement conditional fenced completion in memory, NDJSON, PostgreSQL, and DynamoDB adapters.
- [x] **2.3** Run focused unit and real PostgreSQL/DynamoDB component validation for lease ownership, expiry, stale completion, success, retry, and terminal transitions.
- [x] **3.1** Write a failing DynamoDB unit and local-component test with filtered/ineligible leading records and eligible records beyond the first evaluated page.
- [x] **3.2** Implement bounded DynamoDB pagination that fills the requested eligible batch or exhausts the index without duplicate claims.
- [x] **3.3** Run focused DynamoDB unit, component, and end-to-end validation.
- [x] **4.1** Write failing concurrent-acceptance tests for identical replay and changed-payload conflict against DynamoDB Local and PostgreSQL.
- [x] **4.2** Implement DynamoDB conditional-race rereads and bounded PostgreSQL concurrency retry/reclassification.
- [x] **4.3** Confirm concurrent duplicate intake produces one acceptance plus deterministic replay/conflict outcomes without expected `503` responses.
- [x] **5.1** Write failing readiness tests proving an unavailable selected store returns not-ready while a healthy store returns ready.
- [x] **5.2** Add a bounded readiness port to applicable stores and wire it into production API composition without making Kafka an intake readiness dependency.
- [x] **5.3** Run handler, runtime, local smoke, and storage-backed end-to-end readiness checks.
- [x] **6.1** Write failing deployment contract tests for the missing topic-bootstrap broker/credential configuration and dangling Kubernetes Secret references.
- [x] **6.2** Extend topic-bootstrap configuration to use the bounded AWS/local secret-provider abstraction through IRSA and remove the undeclared Kubernetes Secret dependency.
- [x] **6.3** Make standalone and Terraform-rendered topic bootstrap configuration complete, ordered, externally populated, and locally testable with secure Kafka.
- [x] **6.4** Run topic-bootstrap unit, secure Kafka component, infrastructure, and rendered-manifest validation.
- [x] **7.1** Write failing container/CI contract tests proving the documented artifact path supports the Linux ARM64 architecture selected by Terraform.
- [x] **7.2** Add a deterministic multi-platform build/verification command and CI check, retaining immutable digest deployment requirements.
- [x] **7.3** Validate the ARM64 production binaries/images locally without requiring an AWS account.
- [x] **8.1** Write failing semantic contract tests for OpenAPI validity, documented `405`/`415` responses, and request media-type enforcement.
- [x] **8.2** Enforce `application/json`, document `405` and `415`, and replace the grep-only OpenAPI gate with parser-backed semantic validation.
- [x] **8.3** Extend Kubernetes/Terraform validation to detect missing referenced Secrets, required environment omissions, placeholder deployment inputs, and selector/reference inconsistencies.
- [x] **8.4** Run contract, infrastructure, documentation, and end-to-end validation for the strengthened gates.
- [x] **9.1** Update affected ADRs or add a dated ADR covering fenced outbox ownership, paginated DynamoDB dispatch, dependency-aware readiness, direct secret loading, and deployment architecture compatibility.
- [x] **9.2** Update README, infrastructure/testing/security documentation, behavior matrices, and every affected `AGENTS.md` element description so claims match actual behavior.
- [x] **9.3** Replace the prior release-pass assertion with an evidence-based follow-up review listing every original finding, remediation, residual risk, and local proof.
- [x] **9.4** Turn README onboarding into a Make-driven operator guide covering prerequisites, API startup, local credentials, requests, modes, validation layers, cleanup, and troubleshooting; make direct API startup work with explicit non-production defaults.
- [x] **10.1** Run focused tests first for every change and record the expected red result followed by green evidence in this plan.
- [x] **10.2** Run `make coverage` and confirm every production Go package remains at or above 85% statement coverage.
- [x] **10.3** Run `make validate`, including race, component, end-to-end, infrastructure, security, image, documentation, and hierarchy gates, and confirm cleanup leaves no repository-owned containers, volumes, networks, ports, or generated tracked files.
- [x] **10.4** Reconcile every numbered checklist item individually against evidence and leave no mixed or implicitly completed item.
- [x] **10.5** Move this completed plan from `docs/plans/current/` to `docs/plans/archive/`, update both plan-directory `AGENTS.md` files, rerun documentation/hierarchy/diff validation, and confirm no active plan remains.

## Execution evidence

- **Outbox fencing (2.1–2.3):** focused application and adapter tests first failed against event-ID-only completion. Claim tokens and `ErrLeaseLost` then passed memory, NDJSON, PostgreSQL, DynamoDB, and shared real-store lease/retry/terminal contracts.
- **DynamoDB progress and races (3.1–4.3):** multi-page and concurrent-acceptance tests failed before pagination/reread logic, then passed unit, DynamoDB Local, PostgreSQL, component, and end-to-end paths. Identical concurrent input produces one acceptance and deterministic replay; changed content is a conflict.
- **Readiness (5.1–5.3):** unavailable-store tests failed against constant readiness, then passed with one-second bounded selected-store checks in handler, runtime, smoke, and storage-backed E2E validation. Kafka remains outside intake readiness.
- **Bootstrap/deployment (6.1–6.4):** deployment contracts first exposed missing broker/credential inputs and dangling Secret references. Direct bounded local/AWS secret loading, IRSA, two-stage Terraform gating, namespace/service-account resources, secure Kafka, manifest parsing, and infrastructure checks now pass.
- **ARM64 (7.1–7.3):** container contracts failed before `build-arm64`, `image-arm64`, `TARGETARCH`, and CI coverage existed. Both Linux ARM64 binaries inspect as AArch64 and the loaded production API image reports `arm64` locally.
- **Contracts (8.1–8.4):** semantic OpenAPI/media-type checks failed before `405`/`415` documentation and enforcement. Parser-backed OpenAPI, rendered Kubernetes reference/selector/env/resource/image checks, Terraform contracts, documentation, and E2E gates pass.
- **Documentation (9.1–9.4):** the dated ADR, infrastructure/testing/security records, affected `AGENTS.md` files, command-first README, complete environment reference, and `.env*.example` templates match the implementation. A temporary trusted dotenv started the Go API on `127.0.0.1:18081`; health returned `200` and authenticated intake using the file's key returned `201`, after which the fixture was removed.
- **Repeatability finding:** an interrupted aggregate run left the named memory stack active, causing the next smoke's first request to return a valid replay `200`. A new static contract failed first; splitting pre-start stack cleanup from final evidence cleanup then made `make smoke` pass from stale and clean states.
- **Coverage (10.2):** `make coverage` passed every production package at or above 85%: topic-bootstrap 85.1%, transaction-service 87.1%, bootstrap 93.6%, secrets 86.5%, DynamoDB 88.5%, HTTP 96.3%, Kafka 86.0%, memory 87.3%, NDJSON 91.5%, PostgreSQL 87.2%, app 89.1%, and domain 95.5%.
- **Aggregate validation (10.1–10.4):** `make validate` exited 0 on 2026-08-17 across unit/contract, coverage, component, all local E2E implementations, Terraform, Kubernetes, vulnerability, secret, IaC, and image gates. The subsequently added required `test-race` dependency first hit a Codex sandbox Go-cache/package-resolution failure; the identical `make test-race` command passed unsandboxed in all packages, confirming an execution-environment limitation rather than a repository defect. Documentation links, the 71-directory instruction hierarchy, and `git diff --check` passed. Docker inventory afterward contained no repository-owned containers, volumes, or networks.
- **Preservation:** the pre-existing user edit in `docs/retrospective/2026-08-17-repository-initiation.md` remains untouched and separately visible in the working tree.
