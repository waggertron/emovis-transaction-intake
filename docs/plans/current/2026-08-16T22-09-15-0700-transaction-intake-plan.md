# Transaction Intake Microservice

## Decision Notes

- **User — Initialize version control first.** The first work item is to initialize this repository as Git.
- **User — Write tests before production code.** Every service behavior begins with focused test code that is run and shown to fail; write the smallest implementation, then rerun it to pass before refactoring.
- **User — Require the repository-local `$tdd-development` skill.** Use it for all planning and implementation so test-first acceptance criteria stay explicit.
- **User — Require linked directory-level agent instructions.** Every repository-owned directory must contain an `AGENTS.md` that defines its rules, usage, direct elements, and each element's behavior, and references the applicable parent and child `AGENTS.md` files.
- **User — Make every plan section testable and itemized.** Add explicit unit-test, validation, and smoke-test checklist items for every applicable section; use local mocks by default and treat cloud checks as optional follow-up evidence.
- **User — Confirm everything locally.** Every acceptance criterion, including cloud-oriented behavior, must have repeatable local evidence; real AWS deployment is optional follow-up validation, never a completion dependency.
- **User — Make local operation strict and containerized.** Provide a Makefile as the sole documented command interface, with explicit targets for each system element and validation stage. Every runnable component must have a Dockerfile build target, and Docker Compose must be able to run the complete local system together.
- **User — Make extensive architecture documentation discoverable from the repository entry point.** Add an extensive architectural description to the top-level `README.md` before final handoff.
- **User — Require a final security review.** Perform an evidence-based security review of the completed Go service, dependencies, containers, CI, local environment, and cloud infrastructure; remediate release-blocking findings before handoff.
- **User — Require passing tests for every local external-service implementation.** Kafka, PostgreSQL, DynamoDB Local, the local Secrets Manager substitute, and the combined cloud-equivalence path each require an explicit Make-backed local test that passes before handoff; deterministic client fakes remain necessary unit tests but do not replace these integration checks.
- **User — Require real unit and component coverage for each implementation.** Measure every production Go package separately and exercise every external-service implementation through its real local substitute. Coverage evidence must identify untested critical behavior; aggregate statement percentages, mocks, schema greps, and configuration-only assertions cannot substitute for behavioral component tests.
- **User — Close with checklist confirmation and repository handoff.** The final work item confirms every checklist item is complete, then sends the public repository URL.
- **User — Archive the completed plan last.** After all work is confirmed and the repository URL is shared, move this timestamped plan from `docs/plans/current/` to `docs/plans/archive/`.
- **User — Define a complete checklist first.** Maintain the full work checklist before product implementation.
- **Joint — Mock the missing API contract.** No source OpenAPI spec was supplied; create and clearly label a repository-owned mock OpenAPI contract.
- **User — Deliver a production-shaped toll-event intake service.** Accept one billable toll event per request, with validation, idempotency, documentation, Docker, CI, and tests.
- **Codex recommendation, accepted by User — Use API-key authentication and idempotent replay.** Identical retries return the original acceptance; changed content under the same ID returns a conflict.
- **User — Use a portable ports-and-adapters architecture.** Business logic depends on storage and publisher contracts, not specific infrastructure.
- **User — Provide four interchangeable storage adapters.** In-memory, append-only NDJSON file, DynamoDB, and RDS PostgreSQL all implement the same `TransactionStore` contract. Each deployment selects one adapter; this assignment includes no data migration, replication, or dual-write behavior between stores.
- **User — Use an append-only NDJSON local store.** It is single-process only; configuration rejects separate API/worker file-store operation.
- **Codex recommendation, accepted by User — Use one cohesive `TransactionStore`.** It atomically accepts a transaction and creates its outbox event, then manages outbox claiming and outcomes.
- **User — Use a transactional outbox and separate worker command.** The API and dispatcher deploy separately in cloud environments.
- **User — Use Kafka only for production event delivery.** This supersedes the prior SNS/SQS decision.
- **User — Use Docker Compose locally, EKS for workloads, and Amazon MSK for Kafka.**
- **Codex recommendation, accepted by User — Use `kafka-go`.** Kafka events are at-least-once, keyed by `partnerId:transactionId`, and carry a stable outbox event ID.
- **User — Use bounded, configurable worker retries.** Defaults: five attempts, one-second exponential base delay, one-minute cap, then terminal failed state.
- **User — Provision both DynamoDB and RDS PostgreSQL.** EKS selects the active store through configuration.
- **Codex recommendation, accepted by User — Use `pgx` through `database/sql` and `sqlmock` for test-first PostgreSQL adapter development.**
- **Codex recommendation, accepted by User — Use MSK TLS + SASL/SCRAM credentials from AWS Secrets Manager.**
- **User — Maintain all decisions in the current plan.** This plan begins with Decision Notes; a repo skill will append every later material decision.

## Architecture

```text
Partner
  │ HTTPS + API key
  ▼
API command ──► Transaction application ──► TransactionStore
                    │                         ├─ memory
                    │                         ├─ NDJSON file
                    │                         ├─ DynamoDB
                    │                         └─ PostgreSQL / database/sql
                    │
                    └─ atomically creates pending OutboxEvent

Outbox-worker command ──► claim/retry OutboxEvent ──► Kafka Publisher
                                                      ├─ local Compose Kafka
                                                      └─ Amazon MSK
```

`TransactionStore` owns these operations: accept-with-outbox, claim pending events with a lease, mark publication success, and record a failed attempt. The acceptance operation is atomic in each adapter. The worker uses a 30-second lease and batch size of 10; an expired lease makes a previously claimed event eligible again, which is why consumers deduplicate by event ID.

Kafka publishes `transaction.review-candidates.v1` events with `eventId`, event type/version, occurred time, partner ID, transaction ID, and validated transaction payload. A test-first `topic-bootstrap` command configures the topic idempotently with three partitions and seven-day retention; Docker Compose runs the same command locally and Terraform deploys it as an EKS Job after MSK is available.

## Work Checklist

### Cross-cutting test and validation rules

- [ ] **0.1** Record the focused test name, its expected initial failure, implementation change, and passing rerun for every production behavior; retain this red-green evidence instead of attempting to rerun historical pre-implementation failures from the finished repository.
- [ ] **0.2** Keep unit tests deterministic: inject clocks, IDs, storage, Kafka clients, and configuration sources; do not require network, cloud credentials, or running containers.
- [ ] **0.3** Maintain separate, named valid and invalid fixtures; test fixture validity itself so helpers cannot conceal contract violations.
- [ ] **0.4** Run unit tests before adapter, integration, smoke, security, or manual checks; none of those later checks substitute for the initial failing unit test.
- [ ] **0.5** Use a local/mock adapter for every external boundary. Do not require an AWS account, cloud credentials, or a remote environment for product test, validation, smoke-check, or acceptance evidence; GitHub publication and hosted CI verification are separate external handoff checks.
- [ ] **0.6** Provide a Dockerized local equivalent or deterministic fake for every cloud-oriented adapter: DynamoDB, PostgreSQL/RDS, Secrets Manager, Kafka/MSK, and Kubernetes/EKS configuration.
- [ ] **0.7** Preserve all test logs, Compose artifacts, and temporary data under self-cleaning test paths; do not leave containers, volumes, ports, or fixture files after validation.
- [ ] **0.8** Treat `AGENTS.md` as required repository documentation for every authored directory; exclude only Git internals, dependency/vendor directories, generated caches, and test/runtime artifacts.
- [ ] **0.9** Treat the Makefile as the canonical local command interface. Its targets must fail fast, avoid hidden interactive behavior, print actionable errors, and be safe to run repeatedly; no README, CI, or `AGENTS.md` command may bypass it for a supported workflow.
- [ ] **0.10** Generate one canonical coverage profile through `make coverage`; enforce at least 85% statement coverage independently for every production Go package, including command packages, and fail if any package falls below the threshold rather than allowing high-coverage packages to hide a weak package in an aggregate percentage.
- [ ] **0.11** Treat the percentage as a floor, not proof of correctness. Maintain a reviewed behavior-to-test matrix covering validation/error branches, idempotency, atomic persistence, lease races/expiry, retry/terminal transitions, authentication, body limits, redaction, graceful shutdown, and dependency failures; every critical behavior requires an explicit assertion even when the numerical threshold already passes.
- [ ] **0.12** Separate deterministic unit coverage from component coverage. Unit coverage runs without network or containers; component coverage uses the production adapter against the real local substitute and proves observable behavior and failure handling rather than only checking configuration, SQL strings, fake-client calls, or file contents.
- [ ] **0.13** Publish human-readable per-package and per-component coverage evidence under a self-cleaning local artifact path and summarize the final results in the plan/security review; do not commit generated raw coverage files.

### 1. Repository and decision foundation

- [x] **1.1** Initialize Git and set default branch `main`.
- [x] **1.2** Create this timestamped plan under `docs/plans/current/` with these Decision Notes and checklist.
- [x] **1.3** Create and validate the repository-local `$tdd-development` skill; require it for all planning and implementation work.
- [ ] **1.4** Create and validate `$record-plan-decisions` under `.codex/skills/`; append every material user, Codex, or joint decision with date, attribution, rationale, impact, and supersession.
- [ ] **1.5** Define the `AGENTS.md` template: purpose, scope, local rules, usage, validation commands, an `## Elements` table that names every direct repository-owned child and explains its behavior, and a `## Instruction hierarchy` section.
- [ ] **1.6** Require `## Instruction hierarchy` to link to the nearest parent `AGENTS.md` and every immediate child-directory `AGENTS.md`; the root file explicitly states that it has no parent.
- [ ] **1.7** Add a root `AGENTS.md` requiring decision recording, test-first development, local-only confirmation, and compliance with the directory-level instruction policy.
- [ ] **1.8** Add an `AGENTS.md` to every authored directory in the final repository, including commands, internal packages, adapters, tests, docs, deployment, infrastructure, CI, and repository-local skills.
- [ ] **1.9** Specify the agent-instruction validator's acceptance cases before implementation: missing instructions, missing required sections, malformed or incomplete `## Elements` tables, unresolved links, and parent/child links that do not agree in both directions.
- [ ] **1.10** Add a manual behavior-confirmation checklist: review each `AGENTS.md` against its directory, verify every named element's behavior is accurate, and record the reviewer/date before marking the directory complete.
- [ ] **1.11** Add ADRs for the portable storage/outbox architecture and Kafka/MSK delivery choice.
- [ ] **1.12** Validate every repository skill with `quick_validate.py`; verify its trigger description and default prompt match its instructions.
- [ ] **1.13** Validate the plan and documentation with a Markdown/link check and `git diff --check`.
- [ ] **1.14** Smoke-test a fresh repository checkout after the full instruction hierarchy is present: confirm every authored directory includes a valid `AGENTS.md`, and that plans, skills, code, deployment, and infrastructure instructions are discoverable without generated or local-only artifacts.
- [ ] **1.15** Define the Makefile target contract before implementation: `help`, `test`, `test-unit`, `lint`, `format-check`, `vet`, `build`, `run-api`, `run-worker`, `run-local`, `compose-up`, `compose-down`, `smoke`, `validate`, and `clean`. Document target inputs, outputs, prerequisites, cleanup behavior, and which targets may start containers.

### 2. Tests before implementation

- [ ] **2.1** Initialize only the Go module and test harness.
- [ ] **2.2** For every behavior, run the newly written focused test and confirm its expected failure before adding production code; rerun it after the smallest implementation and confirm it passes.
- [ ] **2.3** Write failing domain tests for IDs, timestamps, toll amount, currency, location, vehicle class, and normalized payload fingerprinting.
- [ ] **2.4** Write failing application tests for acceptance, identical replay, changed duplicate conflict, store failure, atomic outbox creation, claim lease, retry, terminal failure, and consumer-safe event IDs.
- [ ] **2.5** Write a reusable `TransactionStore` contract suite and run it against memory, NDJSON, DynamoDB-client fake, and PostgreSQL/sqlmock adapters.
- [ ] **2.6** Write failing HTTP tests for all documented success/error responses, health, readiness, metrics, and API-key authentication.
- [ ] **2.7** Write failing Kafka publisher tests for key, event envelope, TLS/SASL configuration mapping, and publish failure handling.
- [ ] **2.8** Write failing tests for `topic-bootstrap`: topic creation, idempotent re-run, partition/retention configuration, missing broker configuration, and safe error reporting.
- [ ] **2.9** Write failing tests for the agent-instruction validator: coverage, required sections, parsed `## Elements` table coverage, parent links, child links, reciprocal links, and broken relative links.
- [ ] **2.10** Add explicit valid and invalid fixtures; invalid fixtures must never be repaired by test helpers.
- [ ] **2.11** Add tests for the test helpers/factories themselves, proving valid values stay within contract bounds and invalid modes remain invalid.
- [ ] **2.12** Validate test isolation: run the unit suite repeatedly and verify no test depends on execution order, wall-clock time, network, Docker, or ambient cloud credentials.
- [ ] **2.13** Smoke-test the test harness from a clean module cache/checkout and verify the first planned test fails before production code exists.
- [ ] **2.14** Write failing command-contract tests that verify every required Make target exists, `help` documents it, and the documented target-to-command mapping does not bypass the Makefile.
- [ ] **2.15** Write failing container-definition tests that verify Docker build targets exist for the API, worker, combined-local service, and topic bootstrapper, and that Compose declares a complete local system with no undeclared external dependency.
- [ ] **2.16** Add failing tests for the coverage gate itself: missing package data, any production package below 85%, excluded production source, stale profiles, and a false aggregate pass where one package remains below threshold.
- [ ] **2.17** Add focused tests that raise `cmd/topic-bootstrap` and `cmd/transaction-service` to the same 85% per-package floor, including startup composition, dependency-construction failures, worker cancellation, server shutdown, and safe error paths; do not omit command code from the profile.
- [ ] **2.18** Create the reviewed behavior-to-test matrix and reconcile it against test names and component scenarios; add a missing behavioral test before marking any package covered.

### 3. Contract and service code

- [ ] **3.1** Create the labeled mock OpenAPI 3.1 contract for `POST /v1/transactions`, `/healthz`, `/readyz`, and `/metrics`.
- [ ] **3.2** Define request fields: UUID transaction ID, RFC 3339 occurred-at, positive minor-unit amount, three-letter uppercase currency, agency/plaza/lane IDs, and vehicle class.
- [ ] **3.3** Define responses: `201` new acceptance; `200` and `Idempotent-Replay: true` for identical replay; `409` conflict; `400` malformed JSON; `401` invalid API key; `422` semantic invalidity.
- [ ] **3.4** Implement only enough Go code to satisfy the prewritten tests: domain validation, application use cases, HTTP adapter, request IDs, structured logs, configuration, and graceful shutdown.
- [ ] **3.5** Implement memory, NDJSON, DynamoDB, and PostgreSQL storage adapters behind `TransactionStore`.
- [ ] **3.6** Implement Kafka publisher with `kafka-go`; use a deterministic mock publisher for unit tests.
- [ ] **3.7** Implement `topic-bootstrap` only enough to pass its prewritten tests; use it from local Compose and as the post-MSK Terraform/EKS bootstrap Job.
- [ ] **3.8** Implement the agent-instruction validator only enough to pass its prewritten tests; parse Markdown tables and resolve links rather than searching for arbitrary text.
- [ ] **3.9** Implement API, worker, and combined-local commands from one composition root.
- [ ] **3.10** Refactor only while the full suite remains green.
- [ ] **3.11** Validate the OpenAPI document against its examples and HTTP contract tests: every documented request, status, header, and error body must match live handler behavior.
- [ ] **3.12** Validate each `TransactionStore` adapter with the shared contract suite, including atomic accept-plus-outbox behavior, duplicate comparison, leases, retries, and terminal failures.
- [ ] **3.13** Validate security behavior with tests for missing/incorrect API keys, secret-redacted logs, malformed input limits, request IDs, and no PII in Kafka envelopes.
- [ ] **3.14** Add in-process HTTP smoke tests for `/healthz`, `/readyz`, `/metrics`, new acceptance, identical replay, changed duplicate conflict, and a non-blocking review failure.
- [ ] **3.15** Run static validation after each green suite: `go test ./...`, race-enabled tests where applicable, `go vet ./...`, and formatting checks.
- [ ] **3.16** Implement the Makefile only enough to pass the prewritten command-contract tests. Route all supported build, test, run, Compose, smoke, validation, and cleanup workflows through explicit strict targets.
- [ ] **3.17** Implement Dockerfile build targets and Compose wiring only enough to pass the prewritten container-definition tests; build API, worker, combined-local, and topic-bootstrap images reproducibly from the repository.

### 4. Local Kafka environment

- [ ] **4.1** Add Docker Compose with a single-node KRaft Kafka broker and one combined API/dispatcher service.
- [ ] **4.2** Support local memory storage for fast runs and NDJSON storage for restart/replay demonstration; keep file mode single-process.
- [ ] **4.3** Configure the local worker to publish review-candidate events to Compose Kafka.
- [ ] **4.4** Add Compose smoke tests for valid acceptance, idempotent replay, outbox publication, and inspectable Kafka event output.
- [ ] **4.5** Add a PostgreSQL Compose test profile solely for the PostgreSQL adapter integration suite.
- [ ] **4.6** Validate Compose definitions with `docker compose config --quiet` before starting containers.
- [ ] **4.7** Add a Kafka integration test that consumes the published event and verifies topic, stable key, event ID, schema version, safe payload, and duplicate-safe retry behavior.
- [ ] **4.8** Add a secure-Kafka Compose profile using TLS and SASL/SCRAM test credentials so the MSK connection posture is smoke-tested locally without AWS secrets.
- [ ] **4.9** Add a PostgreSQL integration test that verifies schema migration, unique partner/transaction constraint, SQL transaction rollback, and lease/claim concurrency against the Compose database.
- [ ] **4.10** Add a DynamoDB Local Compose profile and run the shared `TransactionStore` contract suite against the real DynamoDB adapter locally.
- [ ] **4.11** Smoke-test memory and NDJSON combined-local modes independently, including restart/rebuild behavior for NDJSON and configuration rejection for unsafe separate file-store roles.
- [ ] **4.12** Verify teardown removes this stack's containers, volumes, network, temporary files, and ports; assert `docker ps` is clean afterward.
- [ ] **4.13** Verify `make compose-up` starts the complete local system and `make compose-down` removes only this system's resources; verify `make smoke` exercises the documented transaction path through Compose.
- [ ] **4.14** Verify `make run-api`, `make run-worker`, and `make run-local` each select the intended executable mode with explicit configuration and do not silently substitute a different component.
- [ ] **4.15** Add `make test-component-kafka` and prove topic bootstrap, production publisher serialization/keying, real broker publication/consumption, idempotent intake producing one event, retry-safe duplicate delivery, and broker-unavailable failure behavior against local Kafka.
- [ ] **4.16** Add `make test-component-kafka-secure` and prove the production client connects with TLS plus SASL/SCRAM, rejects absent/incorrect credentials and untrusted certificates, publishes/consumes successfully with valid credentials, and cleans generated test certificates and credentials.
- [ ] **4.17** Add `make test-component-postgres` and run the production PostgreSQL adapter against Compose PostgreSQL, proving schema bootstrap, atomic accept-plus-outbox, replay/conflict, rollback, concurrent lease exclusion, lease expiry, retry, terminal failure, publication, restart persistence, and database-unavailable behavior.
- [ ] **4.18** Add `make test-component-dynamodb` and run the production DynamoDB adapter against DynamoDB Local, proving table/GSI bootstrap, transactional accept-plus-outbox, replay/conflict, conditional-write races, conditional lease exclusion, lease expiry, retry, terminal failure, publication/index removal, restart persistence, and service-unavailable behavior.
- [ ] **4.19** Add `make test-component-secrets` and run the production secret-provider abstraction against the local substitute, proving initial lookup, absent/malformed secret rejection, rotation-shaped reload, least-secret exposure, redacted errors/logs, and provider-unavailable behavior.
- [ ] **4.20** Add `make test-component-storage` to run the shared transaction/outbox behavioral contract against memory, NDJSON, real local PostgreSQL, and real DynamoDB Local; record the same behavior matrix for every adapter and permit only explicitly documented capability differences.
- [ ] **4.21** Add `make test-component` as the aggregate local external-service gate. It must run every component target, then the combined cloud-equivalence smoke path, fail if any target was skipped or passed only against a fake, and verify complete teardown afterward.

### 5. AWS cloud environment

- [ ] **5.1** Add Terraform for VPC/networking, EKS, MSK, DynamoDB, RDS PostgreSQL, IAM/IRSA, security groups, and Secrets Manager bindings.
- [ ] **5.2** Deploy the tested `topic-bootstrap` EKS Job after MSK is ready; pass configurable topic name, partition count, and retention through Terraform variables.
- [ ] **5.3** Deploy separate EKS API and worker workloads; configure replica counts, resource requests/limits, readiness/liveness probes, and horizontal-scaling-ready metrics.
- [ ] **5.4** Configure `STORE_DRIVER=dynamodb|postgres`; use IAM for DynamoDB and Secrets Manager-backed PostgreSQL/MSK credentials.
- [ ] **5.5** Configure the worker for MSK TLS/SASL-SCRAM, with no credentials in source, manifests, or logs.
- [ ] **5.6** Document Terraform variables, cloud prerequisites, non-production defaults, and the fact that CI validates but does not apply infrastructure.
- [ ] **5.7** Validate Terraform locally with formatting, initialization, `terraform validate`, and a no-credential mock/example plan. Never apply infrastructure from CI or require AWS credentials for completion.
- [ ] **5.8** Validate Kubernetes manifests with client-side dry-run/schema checks, resource/limit checks, required probes, immutable image references, and no plaintext secrets.
- [ ] **5.9** Unit-test DynamoDB and AWS Secrets Manager adapters with deterministic AWS-client fakes; verify conditional/transactional writes, error classification, secret lookup, and redaction.
- [ ] **5.10** Add a local Secrets Manager substitute selected through the same bootstrap configuration; test secret lookup, rotation-shaped reload behavior, absent-secret failure, and log redaction without AWS credentials.
- [ ] **5.11** Validate PostgreSQL deployment configuration against the same storage contract and confirm the selected `STORE_DRIVER` changes only bootstrap wiring, not application behavior.
- [ ] **5.12** Add a local cloud-equivalence smoke run: use DynamoDB Local, Compose PostgreSQL, local Secrets Manager substitute, and secure Compose Kafka to accept a uniquely identified transaction, persist/claim its outbox event, publish/consume it, and clean all test state.
- [ ] **5.13** Validate MSK/EKS infrastructure locally: TLS-only and SASL/SCRAM configuration mapping, secret-reference policy, least-privilege IAM/IRSA policy documents, topic partition/retention values, broker logging/metric alarm definitions, and Kubernetes workload manifests.
- [ ] **5.14** Document the optional real-AWS smoke procedure separately. It must use only an explicitly selected non-production account and must not be needed to mark any plan item complete.

### 6. Verification and handoff

- [ ] **6.1** Add GitHub Actions for unit tests, contract tests, `go vet ./...`, Docker build, Compose smoke tests, and Terraform formatting/validation.
- [ ] **6.2** Write README material covering the mock-spec limitation, architecture, TDD approach, Makefile command reference, local Compose workflow, configuration, curl examples, storage modes, Kafka event contract, cloud topology, and operations. All runnable instructions must use Make targets.
- [ ] **6.3** Create and push public repository `waggertron/emovis-transaction-intake`; verify hosted README and CI workflow.
- [ ] **6.4** Add CI gates whose product-validation commands are runnable locally without cloud credentials: plan/document validation, skill validation, unit/contract tests, race/vet/format checks, OpenAPI validation, Compose configuration, Docker build, secure-Kafka/DynamoDB-Local/PostgreSQL local smoke tests, Terraform validation, Kubernetes dry-run, and secret scanning.
- [ ] **6.5** Smoke-test the published repository from a fresh clone: install declared dependencies, run the unit suite, start the documented local stack, submit the README example, observe the Kafka event, and perform the documented cleanup.
- [ ] **6.6** Validate repository handoff quality: all links work, no secrets or generated artifacts are tracked, CI is green, Git history contains the plan and skill changes, and every checklist item has recorded evidence.
- [ ] **6.7** Confirm directory-level handoff quality: the agent-instruction validator passes, every authored directory has a completed manual behavior confirmation, and no `AGENTS.md` references missing elements, stale commands, or nonexistent parent/child instruction files.
- [ ] **6.8** Add extensive architecture documentation to the top-level `README.md`: include a component/dependency diagram; API request, idempotency, persistence, outbox, publish/retry, and consumer-deduplication flows; ports-and-adapters boundaries and responsibilities; API/worker/combined-local commands; all storage-adapter choices and single-store deployment constraint; Kafka topic/envelope/key/delivery semantics; local Compose topology; AWS EKS/MSK deployment shape; configuration and secret boundaries; observability, health, and failure handling; scalability, security, and operational trade-offs; plus explicitly deferred decisions. Validate all links and ensure the description agrees with the OpenAPI contract, ADRs, Makefile targets, tests, and deployment artifacts.
- [ ] **6.9** Add the per-package 85% coverage gate and every named component target to `make validate` and hosted CI. Archive readable coverage summaries for CI runs, report the current 72.2% aggregate baseline and each initial package percentage, and require all unit and component coverage gates to pass from a fresh clone before handoff.
- [ ] **6.10** Perform and publish a final evidence-based security review under `docs/security/`. Review Go and `net/http` hardening, API-key comparison, authorization boundaries, request/body limits, input validation, error and log redaction, SQL parameterization, NDJSON path/permission safety, Kafka/MSK TLS and SASL/SCRAM, secret handling, dependency and module integrity, concurrency/race safety, Docker images and runtime privileges, Compose exposure, CI permissions, Kubernetes security context/network exposure, Terraform IAM/network/encryption controls, and repository secret history. Number every finding and record severity, location with line numbers, evidence, impact, fix, mitigation, and false-positive notes. Run locally reproducible security gates including `govulncheck`, race tests, static analysis, dependency review, container/IaC checks, and secret scanning. Resolve every Critical and High finding; resolve each Medium finding or document its explicit risk acceptance and rationale; rerun affected tests test-first and record clean follow-up evidence before release.
- [ ] **6.11** Confirm every preceding plan checklist item is complete, then send the public repository URL.
- [ ] **6.12** Move this completed timestamped plan from `docs/plans/current/` to `docs/plans/archive/` as the final repository action; update any plan links, validate the archive location, and confirm `docs/plans/current/` contains no completed plan.

## Acceptance Criteria

- Every production behavior has a focused test written and demonstrated failing before its implementation, then demonstrated passing afterward.
- Every applicable plan section has explicit unit-test, validation, and smoke-test evidence that runs locally without cloud credentials. Real cloud checks are supplemental and cannot be required for completion.
- Each storage adapter passes the same transaction-and-outbox contract suite.
- A local Compose run accepts a valid transaction, publishes its review event to Kafka, and handles an idempotent replay correctly.
- Cloud manifests and Terraform support separate EKS API/worker workloads, MSK Kafka, DynamoDB, and RDS PostgreSQL without changing business logic.
- Invalid transactions never reach storage or the outbox; review publishing never changes a valid intake response.
- The final security review has no unresolved Critical or High findings, and every accepted Medium risk includes documented ownership and rationale.
- Every material decision is recorded in the top-level Decision Notes section.
- Every production Go package independently meets the 85% statement-coverage floor and every critical behavior in the reviewed matrix has an explicit assertion.
- Kafka, secure Kafka, PostgreSQL, DynamoDB Local, the local secrets substitute, every storage implementation, and the combined cloud-equivalence flow pass their named Make-backed component tests against real local implementations rather than only mocks.
