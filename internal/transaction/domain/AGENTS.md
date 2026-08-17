# Transaction Domain Instructions

## Purpose
Define pure transaction values and invariants.

## Scope
Applies to the transaction domain package.

## Local rules
- Import no HTTP, database, Kafka, AWS, environment, or framework packages.
- Keep validation deterministic and fingerprinting canonical.
- Add and run focused failing tests before implementation.

## Usage
Construct domain transactions at adapter boundaries and validate before persistence.

## Validation
Run `go test ./internal/transaction/domain` and `go test -race ./internal/transaction/domain`.

## Elements
| Element | Behavior |
| --- | --- |
| `transaction.go` | Defines transaction fields, validation, and canonical fingerprints. |
| `transaction_test.go` | Specifies transaction validation and fingerprint behavior. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
