# Emovis Transaction Intake

A Go microservice that validates and idempotently accepts billable tolling transactions, persists transaction and outbox intent atomically, and publishes versioned review-candidate events to Kafka.

The referenced source OpenAPI document was not present in the supplied files. This repository therefore contains an explicitly labeled [mock OpenAPI 3.1 contract](api/openapi.yaml), chosen to keep the assignment concrete while making the assumption visible.

## Quick start

Prerequisites: Go 1.26+, Docker Desktop with Compose, `make`, and `curl`.

```bash
make test
make smoke
```

`make smoke` builds the images, starts Kafka and the combined-local service, creates the topic, verifies acceptance/replay/conflict and the emitted Kafka event, and removes the entire project stack.

To leave the stack running for exploration:

```bash
make compose-up
curl --fail http://127.0.0.1:8080/healthz
make compose-down
```

Submit a transaction:

```bash
curl --fail-with-body \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: local-development-only-key' \
  --data '{"transactionId":"018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01","occurredAt":"2026-08-16T20:30:00Z","amountMinor":725,"currency":"USD","agencyId":"agency-17","plazaId":"plaza-4","laneId":"lane-2","vehicleClass":"CAR"}' \
  http://127.0.0.1:8080/v1/transactions
```

The first request returns `201`. An identical retry returns `200` with `Idempotent-Replay: true`. Reusing the partner/transaction identity with changed content returns `409`.

## Canonical commands

The Makefile is the documented interface for developers and CI.

| Command | Purpose |
| --- | --- |
| `make help` | List supported commands. |
| `make test` | Run deterministic Go tests. |
| `make lint` | Run formatting and `go vet`. |
| `make build` | Build both commands under ignored `.local/bin/`. |
| `make run-api` | Run the API process. |
| `make run-worker` | Run the dispatcher process. |
| `make run-local` | Run API and dispatcher with one shared local store. |
| `make compose-config` | Validate Compose without starting containers. |
| `make compose-up` / `make compose-down` | Start or remove the named local stack. |
| `make smoke` | Run the self-cleaning end-to-end Kafka flow. |
| `make validate` | Run locally reproducible delivery gates. |
| `make clean` | Remove only repository-owned local artifacts. |

Direct process runs require `API_KEY` and `PARTNER_ID`. Kafka defaults to `localhost:9092` and `transaction.review-candidates.v1`. Secrets are never assigned source-code defaults.

## Architecture

### Components and dependency direction

```text
Partner / roadside system
        │  POST /v1/transactions + X-API-Key
        ▼
HTTP adapter ──► IntakeService ──► TransactionStore
                    │                  ├─ memory
                    │                  ├─ append-only NDJSON (combined-local only)
                    │                  ├─ DynamoDB repository (cloud selection)
                    │                  └─ PostgreSQL repository (cloud selection)
                    │
                    └── atomic pending OutboxEvent
                                      │
Worker / combined-local ──► leased claim ──► Kafka publisher
                                                   ├─ Compose KRaft Kafka
                                                   └─ Amazon MSK
```

Dependencies point inward:

- The domain owns transaction fields, invariants, and canonical fingerprints. It imports no HTTP, storage, Kafka, AWS, or environment packages.
- The application owns intake/outbox use cases and the ports they consume.
- Adapters translate HTTP, persistence, and Kafka representations at the boundary.
- Commands are composition roots: they load configuration, select adapters, and own process lifecycle.

### Intake and idempotency flow

```text
authenticate key → bound/strict JSON decode → domain validation → canonical fingerprint
       → storage atomic accept + pending event
           ├─ new identity             → 201 accepted
           ├─ same identity/fingerprint → 200 replay, original event ID
           └─ same identity/different   → 409 conflict
```

Partner identity comes from the API key, not request JSON. Keys are compared as fixed-length SHA-256 digests using constant-time comparison. Bodies are capped at 64 KiB, unknown fields are rejected, internal dependency errors are not returned to clients, and request IDs are validated before reuse.

The idempotency boundary is `partnerId:transactionId`. The canonical fingerprint covers the complete normalized transaction. A replay returns the original outbox event ID and never creates another transaction or event.

### Transactional outbox and failures

Storage owns one cohesive conversation:

1. Accept the validated transaction and pending event atomically.
2. Claim eligible events in batches of 10 with a 30-second lease.
3. Mark successful publication.
4. Record safe failure state and the next retry time.

Retries use a one-second exponential base delay capped at one minute. Five failed attempts produce a terminal failed event. Expired leases become eligible again, so publication is at least once. Publishing never changes an already valid HTTP acceptance response.

Memory storage is concurrency-safe and ephemeral. NDJSON appends and fsyncs each durable transition before making it visible, uses owner-only file permissions, recovers state on restart, and intentionally supports only the combined process; it is not a multi-process database.

### Kafka event contract

The default topic is `transaction.review-candidates.v1`, configured idempotently with three partitions and seven-day retention. Messages use the stable key `partnerId:transactionId` and contain:

- `eventId`, `eventType`, and `schemaVersion`;
- UTC `occurredAt` and a correlation ID;
- partner and transaction identifiers;
- the validated tolling payload, without API keys, authorization headers, or payment credentials.

`kafka-go` uses required-all acknowledgements, five bounded attempts, synchronous writes, hash partitioning, ten-second read/write timeouts, and disabled automatic topic creation. Consumers must deduplicate by `eventId`.

### Executable modes

- `api`: HTTP intake only. Production deployments pair it with a shared DynamoDB or PostgreSQL store.
- `worker`: outbox dispatch only, using the same selected shared store.
- `local`: one process with one shared memory or NDJSON store plus Kafka dispatch.
- `topic-bootstrap`: a short-lived command that configures the topic before workloads start.

The HTTP server sets read-header, read, write, and idle timeouts plus a 16 KiB header limit. SIGINT/SIGTERM triggers a bounded graceful shutdown.

### Local Compose topology

```text
host :8080 → app (non-root combined-local image)
                    │
                    └── kafka:9092 (single-node KRaft)
                              ▲
                       topic-bootstrap (run once after health)
```

Kafka is intentionally single-node and plaintext only inside the isolated local network. Port `8080` binds to loopback. Compose waits for Kafka health and successful topic bootstrap before starting the app. The smoke test uses a named project and proves teardown leaves no project containers.

### AWS deployment shape

The cloud design uses separate EKS API and worker deployments, Amazon MSK, one selected shared store (DynamoDB or RDS PostgreSQL), IRSA/least-privilege IAM, and Secrets Manager. MSK client traffic uses TLS and SASL/SCRAM; credentials are external, never present in source, manifests, events, or logs. A post-MSK EKS Job runs the same topic-bootstrap behavior.

No cloud account or credentials are required for repository completion. Terraform formatting/validation, policy checks, Kubernetes schema checks, and local equivalents provide the required local evidence; an actual non-production AWS smoke run is supplemental.

### Observability and operations

- `/healthz` reports process liveness; `/readyz` reports dependency readiness.
- `/metrics` exposes Prometheus text format without transaction payloads or secrets.
- Structured logs use request/event identity and safe error classes, not credentials or full payloads.
- Production dashboards should track HTTP rate/error/latency, Kafka publish latency/errors, outbox backlog and oldest age, terminal failures, consumer lag, broker health, partition skew, and storage capacity.
- Capacity scales API and worker replicas separately. Kafka key distribution and hot-partner skew must be reviewed before increasing partitions.

### Security boundaries

- API keys and SASL credentials are supplied through environment/Secrets Manager and redacted from configuration strings.
- The browser-oriented CORS surface is disabled; the service uses header authentication rather than cookies, so CSRF is not applicable.
- Runtime images are distroless and explicitly non-root.
- SQL adapters use parameters; cloud policies and networks are least privilege; events contain only the fields needed for review.
- CI runs race, static, vulnerability, secret, container, and IaC checks. The final review is published under `docs/security/` before handoff.

### Trade-offs and deferred decisions

- The mock API contract requires team confirmation against the real inbound contract.
- At-least-once delivery favors durability over duplicate-free transport; consumers carry deduplication responsibility.
- NDJSON is intentionally not horizontally scalable.
- Multi-region disaster recovery, schema registry adoption, partition growth, data retention/deletion policy, PII classification, and a terminal-event remediation workflow require product and platform decisions.
- Real AWS application and destructive infrastructure operations are outside automated repository validation.

See [portable storage/outbox ADR](docs/adr/2026-08-16-portable-storage-and-outbox.md), [Kafka/MSK ADR](docs/adr/2026-08-16-kafka-msk-delivery.md), and the [active implementation plan](docs/plans/current/2026-08-16T22-09-15-0700-transaction-intake-plan.md).

## Testing approach

Every production behavior follows red-green-refactor: write the focused test, run it to observe the intended failure, add the smallest implementation, rerun focused and affected suites, then refactor while green. Unit tests use injected clocks, IDs, stores, publishers, and authentication. Adapter tests add persistence and mapping confidence; Compose smoke verifies wiring but never replaces unit tests.

Repository-specific coding-agent instructions live in [AGENTS.md](AGENTS.md) and `.codex/skills/`. Every authored directory has its own linked `AGENTS.md` describing scope, elements, behavior, and validation.
