# Component Test Instructions

## Purpose
Run shared production-adapter behavior against realistic local service substitutes.

## Scope
Applies under `tests/component/`.

## Local rules
- Run unit-level memory and NDJSON contracts without containers.
- Require explicit environment configuration for PostgreSQL and DynamoDB Local contracts.
- Use unique test state and clean all rows, tables, files, containers, volumes, and networks.
- Never silently replace a requested real component with a fake.

## Usage
Use the named `make test-component-*` targets; `make test-component` is the aggregate gate.

## Validation
Run the focused contract, then the relevant component target, then aggregate validation.

## Elements
| Element | Behavior |
| --- | --- |
| `store_contract_test.go` | Defines the shared transaction/outbox behavioral contract and always runs it against memory and NDJSON. |
| `postgres.sh` | Starts isolated Compose PostgreSQL, runs its shared production-adapter contract, and removes all project state. |
| `dynamodb.sh` | Starts isolated DynamoDB Local, runs production table bootstrap and the shared store contract, and removes all project state. |
| `secrets.sh` | Runs the production local secret provider against real filesystem lookup, reload, rejection, and redaction behavior. |
| `kafka_secure.sh` | Runs production topic bootstrap and negative authentication checks against TLS/SCRAM Kafka. |
| `kafka.sh` | Runs the production plaintext-local Kafka boundary through topic bootstrap, publication, consumption, and teardown. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
