# Transaction Application Instructions

## Purpose
Orchestrate transaction intake and own the external contracts it consumes.

## Scope
Applies to the transaction application package.

## Local rules
- Depend on domain code and consumer-owned ports only.
- Validate before side effects and preserve atomic transaction/outbox intent.
- Inject clocks, IDs, storage, and publishers for deterministic tests.
- Require opaque claim ownership on every outbox completion; stale workers must fail closed.

## Usage
Call application services from transport adapters and workers.

## Validation
Run `go test ./internal/transaction/app` and its race-enabled form.

## Elements
| Element | Behavior |
| --- | --- |
| `intake.go` | Defines the intake use case, storage port, and raw-passthrough-capable outbox envelope. |
| `intake_test.go` | Specifies intake orchestration, idempotency, validation-before-side-effects, and failure behavior. |
| `outbox.go` | Defines fenced lease-based outbox dispatch and bounded retry policy. |
| `outbox_test.go` | Specifies claim ownership, publication, retry, and terminal-failure policy. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
