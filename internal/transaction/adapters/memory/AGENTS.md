# Memory Store Instructions

## Purpose
Provide deterministic, concurrency-safe local storage.

## Scope
Applies to the memory adapter package.

## Local rules
- Implement the application-owned storage contract exactly.
- Preserve atomic acceptance/outbox and original event identity on replay.
- Copy raw audit byte slices so caller mutation cannot change accepted state.

## Usage
Use for unit tests and credential-free local operation.

## Validation
Run focused tests, shared storage contracts, and the race detector.

## Elements
| Element | Behavior |
| --- | --- |
| `store.go` | Implements concurrency-safe atomic acceptance, readiness, and fenced outbox ownership. |
| `store_test.go` | Specifies atomic acceptance, replay, conflict, fencing, and concurrency. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
