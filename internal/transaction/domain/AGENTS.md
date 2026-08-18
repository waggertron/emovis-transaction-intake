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
| `transaction.go` | Defines contract fields, raw audit passthrough, validation, and number-safe canonical fingerprints. |
| `transaction_test.go` | Specifies validation, raw-passthrough, and canonical fingerprint behavior. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
