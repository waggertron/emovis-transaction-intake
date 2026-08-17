# NDJSON Store Instructions

## Purpose
Provide append-only, restart-persistent local transaction and outbox storage.

## Scope
Applies to the NDJSON adapter package.

## Local rules
- Restrict use to one combined process and create files with owner-only permissions.
- Append durable state transitions before exposing them in memory.
- Recover pending and retry state after restart; leases may expire on restart for at-least-once delivery.

## Usage
Use only for combined-local mode with a path under ignored local state.

## Validation
Run focused persistence tests and the race detector using temporary directories.

## Elements
| Element | Behavior |
| --- | --- |
| `store.go` | Implements durable append-before-visible transaction and outbox transitions. |
| `store_test.go` | Specifies restart persistence, idempotency, and outbox lifecycle. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
