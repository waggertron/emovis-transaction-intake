# Emovis Transaction Intake

A Go microservice that validates and idempotently accepts billable tolling transactions, persists transaction and outbox intent atomically, and publishes versioned review-candidate events to Kafka.

The referenced source OpenAPI document was not present in the supplied files. This repository therefore contains an explicitly labeled [mock OpenAPI 3.1 contract](api/openapi.yaml), chosen to keep the assignment concrete while making the assumption visible.

## Developer runbook

### Prerequisites

Install Go 1.26 or newer, Docker Desktop with Compose and Buildx, GNU Make, Git, and `curl`. Terraform and `kubectl` are additionally required for the infrastructure gates. Confirm the core tools before doing anything else:

```bash
go version
docker version
docker compose version
docker buildx version
make --version
curl --version
make help
```

No AWS account, cloud credentials, or real API key is needed for local development. The supported interface is the Makefile; its recipes carry strict shell settings and match CI.

### Start the complete local system

This is the recommended path because it starts Kafka, creates the topic, builds the application, waits for dependencies, and leaves the system running:

```bash
make compose-up
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

The local credential is intentionally public test data:

```text
partner ID: local-partner
API key:    local-development-only-key
```

You do not obtain or generate this key. It is a repository sentinel used only by Compose, Make, tests, and examples; never deploy it. Production keys are provisioned outside Git and loaded from AWS Secrets Manager or another approved secret-delivery system.

Submit a transaction:

```bash
curl --fail-with-body \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: local-development-only-key' \
  --data '{"transactionId":"018f47a8-40d1-7e32-b6d6-4f4f8f9c9e01","occurredAt":"2026-08-16T20:30:00Z","amountMinor":725,"currency":"USD","agencyId":"agency-17","plazaId":"plaza-4","laneId":"lane-2","vehicleClass":"CAR"}' \
  http://127.0.0.1:8080/v1/transactions
```

The first request returns `201`. Sending that exact request again returns `200` and `Idempotent-Replay: true`. Reusing the same partner/transaction identity with different content returns `409`. Inspect metrics with `curl http://127.0.0.1:8080/metrics`.

Always stop the stack when finished:

```bash
make compose-down
```

`compose-down` removes this repository's containers, network, and volumes. `make clean` additionally removes repository-owned build and test output.

### Start only the API

For fast HTTP work that does not need Kafka publication, run:

```bash
make run-api
```

This uses the in-memory store and the same local partner/key shown above. It listens on `:8080`; stop it with Ctrl-C. Override the safe Make fallbacks without editing files when a populated `.env` does not supply them:

```bash
make run-api LOCAL_PARTNER_ID=partner-demo LOCAL_API_KEY=replace-for-local-testing
```

Then send `X-API-Key: replace-for-local-testing`. This process does not publish outbox events because no worker is running, and its in-memory data disappears on exit.

`make run-worker` starts only the dispatcher. In practice, an independently started API and worker must use the same PostgreSQL or DynamoDB store and a reachable Kafka cluster; separate processes cannot share the memory or NDJSON implementations. `make run-local` combines API and worker in one process, but it still expects Kafka at `localhost:9092`. Use `make compose-up` unless you deliberately manage those dependencies yourself.

### Run tests and delivery gates

Use the smallest relevant command while developing, then the aggregate gate before handoff:

| Command | What it proves |
| --- | --- |
| `make test` / `make test-race` | Go unit and contract behavior, then concurrent safety under the race detector. |
| `make test-contract` | Make, OpenAPI, container, CI, coverage, and instruction-hierarchy contracts. |
| `make coverage` | Every production Go package has at least 85% statement coverage. |
| `make lint` | Formatting and `go vet` pass. |
| `make build` | Both native commands build under `.local/bin/`. |
| `make build-arm64` | Both Linux ARM64 commands cross-compile and inspect correctly. |
| `make image-arm64` | The production API image builds and reports the ARM64 architecture used by Graviton nodes. |
| `make compose-config` | The complete Compose model parses without starting services. |
| `make smoke` | A self-cleaning memory → outbox → Kafka acceptance/replay/conflict flow passes. |
| `make test-component` | PostgreSQL, DynamoDB Local, plaintext/secure Kafka, and secret providers satisfy their real adapter contracts. |
| `make test-e2e` | Every local implementation works across HTTP, persistence, worker, and Kafka process boundaries. |
| `make test-cloud-equivalence` | Production-selected external boundaries pass against local equivalents. |
| `make terraform-validate` | All pinned AWS modules validate without applying or choosing a backend. |
| `make terraform-plan TFVARS=dynamodb.tfvars.example` | Plan the shared topology with only the DynamoDB storage module. |
| `make terraform-plan TFVARS=postgres.tfvars.example` | Plan the shared topology with only the PostgreSQL storage module. |
| `make k8s-validate` / `make test-infrastructure` | Rendered Kubernetes and IaC security/reference contracts pass offline. |
| `make security` | Vulnerability, secret, configuration, and production-image scans pass. |
| `make validate-static` | Fast aggregate gates pass, including documentation and agent-instruction validation. |
| `make validate` | Every locally reproducible unit, race, coverage, component, E2E, infrastructure, security, and cleanup-aware gate passes. |
| `make clean` | Repository-owned generated files and Compose resources are removed. |

`make smoke` and the Compose-backed test scripts perform their own teardown, including on failure. Docker-backed aggregates take longer and require the Docker daemon to be running.

### Environment files

[`.env.local.example`](.env.local.example) is safe, runnable local sample data. [`.env.production.example`](.env.production.example) is a production-shaped inventory with placeholders only. Never add real credentials to either file or commit a populated `.env`.

Docker Compose automatically reads a root `.env` for `${...}` interpolation. The direct `make run-api`, `make run-worker`, and `make run-local` commands source the trusted file selected by `ENV_FILE` first, export its values to Go, and fall back to `LOCAL_API_KEY` and `LOCAL_PARTNER_ID` only when those required values remain empty. Copy and edit the local template, then run normally:

```bash
cp .env.local.example .env
make run-api
```

Use another trusted file with `make run-api ENV_FILE=.env.my-local`. The recipe sources the file as shell syntax, so never point `ENV_FILE` at downloaded or untrusted content. To bypass dotenv loading, use `ENV_FILE=/dev/null` and provide the `LOCAL_*` fallbacks.

The complete application and topic-bootstrap configuration surface is below. An empty value means “unset,” not a literal placeholder.

| Variable | Required and used by | Meaning, default, and valid values | Sensitive |
| --- | --- | --- | --- |
| `HTTP_ADDRESS` | Optional; transaction service | Listen address; defaults to `:8080`. | No |
| `PARTNER_ID` | Required in every transaction-service mode | Identity attached to requests authenticated by the configured key. | Usually no |
| `API_KEY` | Required in every transaction-service mode | Exact value expected in `X-API-Key`; no application default. Make/Compose supply the local sentinel. | Yes |
| `STORE_DRIVER` | Optional; transaction service | `memory` (default), `ndjson`, `postgres`, or `dynamodb`. NDJSON is valid only in `local` mode. | No |
| `STORE_PATH` | Optional; NDJSON | File path; defaults to `.local/data/transactions.ndjson`. | No |
| `POSTGRES_URL` | Required only when `STORE_DRIVER=postgres` | PostgreSQL connection URL used by API and worker. | Yes |
| `DYNAMODB_ENDPOINT` | Optional; DynamoDB | Endpoint override for DynamoDB Local; leave empty in AWS. | No |
| `AWS_REGION` | Optional; DynamoDB/AWS providers | AWS region; defaults to `us-west-2`. | No |
| `DYNAMODB_TABLE` | Optional; DynamoDB | Table name; defaults to `transaction-intake`. | No |
| `KAFKA_BROKERS` | Optional for transaction service; required for topic bootstrap | Comma-separated brokers. Transaction service defaults to `localhost:9092`; bootstrap has no broker default. | No |
| `KAFKA_TOPIC` | Optional; transaction service and bootstrap | Topic name; defaults to `transaction.review-candidates.v1`. | No |
| `KAFKA_TLS` | Optional; transaction service and bootstrap | Boolean (`true`/`false`); defaults to `false`. Must be true when SASL or a CA file is configured. | No |
| `KAFKA_CA_FILE` | Optional; transaction service and bootstrap | PEM CA bundle path for TLS verification. | No |
| `KAFKA_SASL_USERNAME` | Optional; transaction service and bootstrap | SASL/SCRAM username; must be paired with password and TLS. | Yes |
| `KAFKA_SASL_PASSWORD` | Optional; transaction service and bootstrap | SASL/SCRAM password; must be paired with username and TLS. | Yes |
| `KAFKA_TOPIC_PARTITIONS` | Optional; topic bootstrap only | Positive integer; defaults to `3`. | No |
| `KAFKA_TOPIC_REPLICATION` | Optional; topic bootstrap only | Positive integer; defaults to `1` locally; use a cluster-appropriate value such as `3` in production. | No |
| `KAFKA_TOPIC_RETENTION` | Optional; topic bootstrap only | Positive Go duration; defaults to `168h` (seven days). | No |
| `LOCAL_SECRET_FILE` | Optional; transaction service and bootstrap | Path to a maximum-64-KiB JSON object supplying otherwise-unset variables. Mutually exclusive with `AWS_SECRET_ID`. | Path no; contents yes |
| `AWS_SECRET_ID` | Optional; transaction service and bootstrap | Secrets Manager name/ARN containing a maximum-64-KiB JSON object. Mutually exclusive with `LOCAL_SECRET_FILE`. | Identifier usually no; contents yes |

Explicit non-empty environment values take precedence over values loaded from a secret provider. A provider payload is a flat JSON string map, for example `{"API_KEY":"...","PARTNER_ID":"...","KAFKA_SASL_USERNAME":"...","KAFKA_SASL_PASSWORD":"..."}`. Do not commit an actual payload. AWS SDK authentication uses its normal credential chain; EKS supplies `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE` through IRSA, so the application does not define or store static AWS access keys.

Make also accepts these workflow controls: `ENV_FILE` selects the trusted dotenv file (default `.env`); `LOCAL_API_KEY` and `LOCAL_PARTNER_ID` are direct-run fallbacks when dotenv does not define them; `TFVARS` makes the required Terraform storage choice; `COMPOSE_PROJECT_NAME` and `COMPOSE_FILE` select the isolated Compose project/model; `GITLEAKS_IMAGE`, `TRIVY_IMAGE`, `SECURITY_IMAGE`, and `ARM64_IMAGE` override pinned scanner or output image names. Test scripts manage their own fixture-only variables, ports, UIDs/GIDs, and service URLs; developers do not need to set them for canonical Make targets.

### Choose Terraform storage explicitly

Terraform intentionally has no default database. The repository retains both production implementations for review and validation, but a plan cannot proceed until you make the architecture decision:

```bash
# Choose DynamoDB
make terraform-plan TFVARS=dynamodb.tfvars.example

# Or choose PostgreSQL
make terraform-plan TFVARS=postgres.tfvars.example
```

Do not run both for one environment. The DynamoDB selection creates only the transaction/outbox table; the PostgreSQL selection creates only the private RDS instance, subnet group, and PostgreSQL secret container. Shared EKS, MSK, networking, encryption, API secret, and Kafka secret resources remain common to either plan. Workload IAM includes DynamoDB access only for the DynamoDB selection.

Running `make terraform-plan` without `TFVARS` exits with an actionable error. This is deliberate: neither Terraform nor Make is allowed to choose a persistence technology on the user's behalf. `make terraform-validate` still validates all module definitions without making a selection, and there is no automated `terraform apply` target. See the [storage infrastructure selection ADR](docs/adr/2026-08-17-explicit-storage-infrastructure-selection.md) for the rationale and trade-offs.

### Troubleshooting

- `make run-api` reports that port 8080 is in use: stop the other process or run `HTTP_ADDRESS=:8081 make run-api`, then call port 8081.
- A request returns `401`: the `X-API-Key` header does not exactly match `LOCAL_API_KEY`/`API_KEY`.
- A request returns `415`: send `Content-Type: application/json`.
- `/healthz` passes but `/readyz` fails: the process is alive but its selected store is unavailable or misconfigured.
- Docker commands cannot connect: start Docker Desktop and rerun `docker version` before the Make target.
- `make run-local` exits on Kafka errors: use `make compose-up`, which starts and bootstraps Kafka, or configure a reachable broker and topic.
- PostgreSQL startup fails: set `STORE_DRIVER=postgres` and a valid `POSTGRES_URL` together.
- DynamoDB Local is not used: set both `STORE_DRIVER=dynamodb` and `DYNAMODB_ENDPOINT`; omit the endpoint for AWS DynamoDB.
- Secret loading fails: set exactly one of `LOCAL_SECRET_FILE` or `AWS_SECRET_ID`, ensure the JSON is a non-empty string map under 64 KiB, and keep explicit environment overrides intentional.
- A Docker-backed test was interrupted: run `make compose-down` followed by `make clean`, then retry.

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

Every claim includes an opaque ownership token. Success and failure updates must present the current token, so a worker that resumes after lease expiry cannot overwrite the replacement worker's result. Retries use a one-second exponential base delay capped at one minute. Five failed attempts produce a terminal failed event. Expired leases become eligible again, so publication is at least once. Publishing never changes an already valid HTTP acceptance response.

DynamoDB follows filtered query pages until the requested eligible batch is filled or candidates are exhausted. DynamoDB conditional races and PostgreSQL uniqueness/serialization races are reread and classified as replay or conflict rather than exposed as ordinary availability failures.

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

The cloud design uses separate EKS API and worker deployments, Amazon MSK, one selected shared store (DynamoDB or RDS PostgreSQL), IRSA/least-privilege IAM, and Secrets Manager. MSK client traffic uses TLS and SASL/SCRAM; credentials are external, never present in source, manifests, events, or logs. A post-MSK EKS Job loads bounded configuration directly from Secrets Manager through IRSA. Terraform enables SCRAM association and that Job only after an operator confirms authorized secret population with `runtime_secrets_ready=true`.

No cloud account or credentials are required for repository completion. Terraform formatting/validation, policy checks, Kubernetes schema checks, and local equivalents provide the required local evidence; an actual non-production AWS smoke run is supplemental. See the [AWS infrastructure reference](docs/infrastructure/reference.md) and [optional non-production smoke procedure](docs/infrastructure/aws-smoke.md).

### Observability and operations

- `/healthz` reports process liveness; `/readyz` performs a one-second bounded check of the selected intake store. Kafka remains asynchronous and is not an API readiness dependency.
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
- The evidence and remediations are recorded in the [final security review](docs/security/2026-08-17-final-review.md).

### Trade-offs and deferred decisions

- The mock API contract requires team confirmation against the real inbound contract.
- At-least-once delivery favors durability over duplicate-free transport; consumers carry deduplication responsibility.
- NDJSON is intentionally not horizontally scalable.
- Multi-region disaster recovery, schema registry adoption, partition growth, data retention/deletion policy, PII classification, and a terminal-event remediation workflow require product and platform decisions.
- Real AWS application and destructive infrastructure operations are outside automated repository validation.

See [portable storage/outbox ADR](docs/adr/2026-08-16-portable-storage-and-outbox.md), [Kafka/MSK ADR](docs/adr/2026-08-16-kafka-msk-delivery.md), and the [archived implementation plan](docs/plans/archive/2026-08-16T22-09-15-0700-transaction-intake-plan.md).

## Testing approach

Every production behavior follows red-green-refactor: write the focused test, run it to observe the intended failure, add the smallest implementation, rerun focused and affected suites, then refactor while green. Unit tests use injected clocks, IDs, stores, publishers, and authentication. Adapter tests add persistence and mapping confidence; Compose smoke verifies wiring but never replaces unit tests.

Repository-specific coding-agent instructions live in [AGENTS.md](AGENTS.md) and `.codex/skills/`. Every authored directory has its own linked `AGENTS.md` describing scope, elements, behavior, and validation.

The [behavior-to-test matrix](docs/testing/behavior-matrix.md) maps critical correctness and failure risks to focused assertions, real local-service components, and end-to-end paths.

## Plain-language onboarding overview

The following explanation is preserved verbatim as a point-in-time onboarding summary. Operational status statements describe the validated handoff recorded in the archived implementation plan; use the current GitHub Actions run for live status.

You built a production-shaped toll transaction intake service.

A roadside system sends the service a toll transaction—something like:

```json
{
  "transactionId": "018f47a8-40d1-7e32-b6d6-4f4f8f9c9e27",
  "occurredAt": "2026-08-17T18:30:00Z",
  "amountMinor": 725,
  "currency": "USD",
  "agencyId": "agency-17",
  "plazaId": "plaza-4",
  "laneId": "lane-2",
  "vehicleClass": "CAR"
}
```

`amountMinor: 725` means `$7.25`. Using minor units avoids floating-point money bugs.

## What happens to a request

```text
Roadside system
      │
      ▼
  HTTP API
      │ validate + authenticate
      ▼
Transaction service
      │
      ├── save transaction
      └── save pending Kafka event
              │
              ▼
       Background worker
              │
              ▼
            Kafka
```

The API:

1. Checks the API key.
2. Validates the JSON and business fields.
3. Saves the transaction.
4. Creates a pending event for Kafka.
5. Returns an HTTP response.

The background worker later sends that event to Kafka, where another system could review, rate, reconcile, or process it.

## Why the database and Kafka event are saved together

A classic failure looks like this:

1. Save the transaction.
2. Try to publish to Kafka.
3. Kafka is temporarily unavailable.
4. The transaction exists, but no event was published.

The service prevents that with a transactional outbox.

The transaction and a pending outbox record are saved atomically—either both succeed or neither succeeds. A worker publishes pending records later and retries failures.

Kafka delivery is at-least-once, meaning an event might occasionally arrive more than once. Each event therefore has a stable ID so consumers can deduplicate it.

## Duplicate requests

Roadside systems retry requests because networks are unreliable.

The service handles retries intentionally:

- First valid request: `201 Created`
- Same ID and same content again: `200 OK` with `Idempotent-Replay: true`
- Same ID but different content: `409 Conflict`

That prevents both accidental double billing and silent transaction replacement.

## Storage choices

The business logic talks to a `TransactionStore` interface instead of directly depending on one database.

There are four implementations:

- Memory: fast, temporary development mode
- NDJSON: append-only local file storage
- PostgreSQL: relational production option
- DynamoDB: AWS-native production option

A deployment selects one store. The service does not write to PostgreSQL and DynamoDB simultaneously.

Every store follows the same behavioral contract: transaction acceptance, replay, conflicts, outbox creation, event claiming, retries, and completion.

## Local versus AWS architecture

Locally, Docker Compose runs substitutes for the external infrastructure:

- Kafka
- PostgreSQL
- DynamoDB Local
- Local secret provider
- API and worker processes

In AWS, the intended shape is:

```text
Internet/partner network
          │
          ▼
       EKS API
          │
    PostgreSQL or DynamoDB
          │
          ▼
      EKS worker
          │
          ▼
        Amazon MSK
```

Terraform defines:

- VPC and private networking
- EKS
- Amazon MSK
- PostgreSQL RDS
- DynamoDB
- Secrets Manager
- IAM and IRSA permissions
- Logging, alarms, encryption, and flow logs

The Terraform configuration can be validated locally without AWS credentials and does not automatically deploy anything.

## Kafka

Kafka receives `transaction.review-candidates.v1` events.

Events are keyed by:

```text
partnerId:transactionId
```

That gives related retries a stable partitioning key.

A topic-bootstrap command creates or configures the topic. The same concept is used in Docker Compose and the EKS deployment.

Both plain local Kafka and a production-shaped secure Kafka mode are tested. The secure mode uses:

- TLS certificate verification
- SASL/SCRAM authentication
- No hard-coded production credentials

## Security

The service includes several practical protections:

- API-key authentication
- Constant-time API-key comparison
- Request body limits
- Strict transaction validation
- Parameterized SQL
- Non-root containers
- Read-only secret mounts
- Private EKS access
- Least-privilege workload identity
- Encrypted storage and network logging
- No secrets in logs or Kafka payloads
- Dependency vulnerability scanning
- Git-history secret scanning
- Infrastructure and container scanning

The initial security review was reopened by a later whole-repository adversarial review. Current remediation status is recorded in the [adversarial follow-up](docs/security/2026-08-17-adversarial-follow-up.md).

## Testing strategy

The repository follows test-driven development:

```text
Write a test
    ↓
Watch it fail
    ↓
Write the smallest implementation
    ↓
Watch it pass
    ↓
Refactor safely
```

Testing is layered:

- Unit tests: business behavior in isolation
- Contract tests: all storage adapters obey the same rules
- Component tests: production adapters against real local services
- E2E tests: HTTP → storage → worker → Kafka
- Smoke tests: prove the documented developer workflow works
- Security tests: vulnerabilities, secrets, containers, and infrastructure
- Static checks: formatting, vetting, documentation links, and repository structure

Every production Go package has at least 85% statement coverage. The project also tests important behavior explicitly, rather than treating coverage percentage as proof that everything works.

## Repository instructions

Every directory has an `AGENTS.md` explaining:

- What belongs there
- How its files behave
- How to validate changes
- Which parent and child instructions apply

The `.codex/skills` directory adds repository-specific guidance for:

- TDD
- Architecture decisions
- Kafka practices
- Engineering technology choices
- Recording decisions in the active plan
- Maintaining the `AGENTS.md` hierarchy

This is the configuration an AI coding agent uses to understand how it should work inside the repository.

## How developers use it

The Makefile is the main interface. Developers do not need to memorize long commands.

Typical commands include:

```bash
make test
make coverage
make test-component
make test-e2e
make smoke
make security
make validate
```

Docker Compose runs the complete local system and cleans it up afterward.

## CI and handoff

GitHub Actions independently runs:

- Quality and coverage
- Infrastructure validation
- Security scanning
- Local smoke testing
- Component testing
- Every local implementation end to end

All six jobs pass on the final commit.

The repository is private:

https://github.com/waggertron/emovis-transaction-intake

Anyone reviewing it needs explicit GitHub access.

In short: it is a small toll-ingestion API, but it demonstrates the architecture and engineering practices expected of a real service—reliable retries, interchangeable storage, event delivery, local cloud substitutes, secure deployment configuration, comprehensive tests, and clear operating documentation.
