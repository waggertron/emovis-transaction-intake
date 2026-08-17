# Transaction Intake Microservice

## Decision Notes

- **User — Initialize version control first.** The first work item is to initialize this repository as Git.
- **User — Write tests before production code.** Every service behavior begins with focused test code; implementation follows red-green-refactor.
- **User — Close with checklist confirmation and repository handoff.** The final work item confirms every checklist item is complete, then sends the public repository URL.
- **User — Define a complete checklist first.** Maintain the full work checklist before product implementation.
- **Joint — Mock the missing API contract.** No source OpenAPI spec was supplied; create and clearly label a repository-owned mock OpenAPI contract.
- **User — Deliver a production-shaped toll-event intake service.** Accept one billable toll event per request, with validation, idempotency, documentation, Docker, CI, and tests.
- **Codex recommendation, accepted by User — Use API-key authentication and idempotent replay.** Identical retries return the original acceptance; changed content under the same ID returns a conflict.
- **User — Use a portable ports-and-adapters architecture.** Business logic depends on storage and publisher contracts, not specific infrastructure.
- **User — Provide four interchangeable storage adapters.** In-memory, append-only NDJSON file, DynamoDB, and RDS PostgreSQL all implement the same `TransactionStore` contract.
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
                    │                         └─ PostgreSQL / database.sql
                    │
                    └─ atomically creates pending OutboxEvent

Outbox-worker command ──► claim/retry OutboxEvent ──► Kafka Publisher
                                                      ├─ local Compose Kafka
                                                      └─ Amazon MSK
```

`TransactionStore` owns these operations: accept-with-outbox, claim pending events with a lease, mark publication success, and record a failed attempt. The acceptance operation is atomic in each adapter. The worker uses a 30-second lease and batch size of 10; an expired lease makes a previously claimed event eligible again, which is why consumers deduplicate by event ID.

Kafka publishes `transaction.review-candidates.v1` events with `eventId`, event type/version, occurred time, partner ID, transaction ID, and validated transaction payload. Terraform provisions three partitions and seven-day retention, exposed as variables.

## Work Checklist

### 1. Repository and decision foundation

- [x] Initialize Git and set default branch `main`.
- [x] Create this timestamped plan under `docs/plans/current/` with these Decision Notes and checklist.
- [ ] Create and validate `$record-plan-decisions` under `.codex/skills/`; append every material user, Codex, or joint decision with date, attribution, rationale, impact, and supersession.
- [ ] Add `AGENTS.md` requiring decision recording and test-first development.
- [ ] Add ADRs for the portable storage/outbox architecture and Kafka/MSK delivery choice.

### 2. Tests before implementation

- [ ] Initialize only the Go module and test harness.
- [ ] Write failing domain tests for IDs, timestamps, toll amount, currency, location, vehicle class, and normalized payload fingerprinting.
- [ ] Write failing application tests for acceptance, identical replay, changed duplicate conflict, store failure, atomic outbox creation, claim lease, retry, terminal failure, and consumer-safe event IDs.
- [ ] Write a reusable `TransactionStore` contract suite and run it against memory, NDJSON, DynamoDB-client fake, and PostgreSQL/sqlmock adapters.
- [ ] Write failing HTTP tests for all documented success/error responses, health, readiness, metrics, and API-key authentication.
- [ ] Write failing Kafka publisher tests for key, event envelope, TLS/SASL configuration mapping, and publish failure handling.
- [ ] Add explicit valid and invalid fixtures; invalid fixtures must never be repaired by test helpers.

### 3. Contract and service code

- [ ] Create the labeled mock OpenAPI 3.1 contract for `POST /v1/transactions`, `/healthz`, `/readyz`, and `/metrics`.
- [ ] Define request fields: UUID transaction ID, RFC 3339 occurred-at, positive minor-unit amount, three-letter uppercase currency, agency/plaza/lane IDs, and vehicle class.
- [ ] Define responses: `201` new acceptance; `200` and `Idempotent-Replay: true` for identical replay; `409` conflict; `400` malformed JSON; `401` invalid API key; `422` semantic invalidity.
- [ ] Implement only enough Go code to satisfy the prewritten tests: domain validation, application use cases, HTTP adapter, request IDs, structured logs, configuration, and graceful shutdown.
- [ ] Implement memory, NDJSON, DynamoDB, and PostgreSQL storage adapters behind `TransactionStore`.
- [ ] Implement Kafka publisher with `kafka-go`; use a deterministic mock publisher for unit tests.
- [ ] Implement API, worker, and combined-local commands from one composition root.
- [ ] Refactor only while the full suite remains green.

### 4. Local Kafka environment

- [ ] Add Docker Compose with a single-node KRaft Kafka broker and one combined API/dispatcher service.
- [ ] Support local memory storage for fast runs and NDJSON storage for restart/replay demonstration; keep file mode single-process.
- [ ] Configure the local worker to publish review-candidate events to Compose Kafka.
- [ ] Add Compose smoke tests for valid acceptance, idempotent replay, outbox publication, and inspectable Kafka event output.
- [ ] Add a PostgreSQL Compose test profile solely for the PostgreSQL adapter integration suite.

### 5. AWS cloud environment

- [ ] Add Terraform for VPC/networking, EKS, MSK, DynamoDB, RDS PostgreSQL, IAM/IRSA, security groups, and Secrets Manager bindings.
- [ ] Provision the Kafka topic with configurable default partitions and retention.
- [ ] Deploy separate EKS API and worker workloads; configure replica counts, resource requests/limits, readiness/liveness probes, and horizontal-scaling-ready metrics.
- [ ] Configure `STORE_DRIVER=dynamodb|postgres`; use IAM for DynamoDB and Secrets Manager-backed PostgreSQL/MSK credentials.
- [ ] Configure the worker for MSK TLS/SASL-SCRAM, with no credentials in source, manifests, or logs.
- [ ] Document Terraform variables, cloud prerequisites, non-production defaults, and the fact that CI validates but does not apply infrastructure.

### 6. Verification and handoff

- [ ] Add GitHub Actions for unit tests, contract tests, `go vet ./...`, Docker build, Compose smoke tests, and Terraform formatting/validation.
- [ ] Write README material covering the mock-spec limitation, architecture, TDD approach, local Compose workflow, configuration, curl examples, storage modes, Kafka event contract, cloud topology, and operations.
- [ ] Create and push public repository `waggertron/emovis-transaction-intake`; verify hosted README and CI workflow.
- [ ] Confirm every plan checklist item is complete, then send the public repository URL.

## Acceptance Criteria

- Every production behavior has a focused test written before its implementation.
- Each storage adapter passes the same transaction-and-outbox contract suite.
- A local Compose run accepts a valid transaction, publishes its review event to Kafka, and handles an idempotent replay correctly.
- Cloud manifests and Terraform support separate EKS API/worker workloads, MSK Kafka, DynamoDB, and RDS PostgreSQL without changing business logic.
- Invalid transactions never reach storage or the outbox; review publishing never changes a valid intake response.
- Every material decision is recorded in the top-level Decision Notes section.
