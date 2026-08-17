# Command Instructions

## Purpose
Contain thin executable composition roots.

## Scope
Applies under `cmd/`.

## Local rules
- Keep business rules out of commands.
- Parse mode/configuration, wire dependencies, handle lifecycle, and redact failures.

## Usage
Build all commands through `make build` and invoke documented modes through Make.

## Validation
Run command tests, builds, and process smoke tests.

## Elements
| Element | Behavior |
| --- | --- |
| `topic-bootstrap` | Idempotently configures the review-candidate Kafka topic. |
| `transaction-service` | Runs API, worker, or combined-local mode. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [topic-bootstrap/AGENTS.md](topic-bootstrap/AGENTS.md).
- Child: [transaction-service/AGENTS.md](transaction-service/AGENTS.md).
