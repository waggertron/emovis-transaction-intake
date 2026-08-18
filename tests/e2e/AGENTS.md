# End-to-End Test Instructions

## Purpose
Exercise every local implementation through production HTTP, storage, outbox, and Kafka process boundaries.

## Scope
Applies to end-to-end scripts under `tests/e2e/`.

## Local rules
- Use a unique Compose project and explicit transaction fixture per test.
- Use production images and adapters; deterministic fakes do not satisfy these tests.
- Verify acceptance, Kafka publication, replay, contract-shaped 400 validation errors, raw numeric passthrough, persistence where applicable, dependency failure, and complete teardown.
- Never retain containers, volumes, networks, ports, credentials, certificates, temporary files, or seeded state.

## Usage
Run an implementation-specific `make test-e2e-*` target or the aggregate `make test-e2e` gate.

## Validation
Each script must exit nonzero on a skipped assertion and verify its Compose resources are absent after teardown.

## Elements
| Element | Behavior |
| --- | --- |
| `lib.sh` | Provides strict HTTP/Kafka assertions and scoped Compose cleanup. |
| `retry_test.sh` | Unit-tests bounded retry behavior for transient Compose startup failures. |
| `memory.sh` | Runs the combined production process with the in-memory adapter. |
| `ndjson.sh` | Runs the combined production process with persistent NDJSON and a restart. |
| `postgres.sh` | Runs separate production API and worker processes with PostgreSQL. |
| `dynamodb.sh` | Runs separate production API and worker processes with DynamoDB Local. |
| `secrets.sh` | Runs the production path using the local secret provider. |
| `kafka_secure.sh` | Runs the production path using TLS and SASL/SCRAM Kafka. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
