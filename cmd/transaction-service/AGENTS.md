# Transaction Service Command Instructions

## Purpose
Compose transaction service dependencies and process lifecycles.

## Scope
Applies to the transaction-service executable.

## Local rules
- Support only `api`, `worker`, and `local` modes.
- Use explicit HTTP limits, graceful shutdown, and externally supplied secrets.
- Keep local combined mode on a shared store.

## Usage
Invoke through `make run-api`, `make run-worker`, or `make run-local`.

## Validation
Run command unit tests, `make build`, and Compose smoke tests.

## Elements
| Element | Behavior |
| --- | --- |
| `main.go` | Parses mode/configuration and selects the exact process starter. |
| `main_test.go` | Specifies mode dispatch and configuration failure behavior. |
| `runtime.go` | Wires HTTP, memory storage, outbox dispatch, Kafka, and graceful lifecycle. |
| `runtime_test.go` | Specifies safe storage-driver selection and local state setup. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
