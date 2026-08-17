# Repository Test Instructions

## Purpose
Contain tests that validate cross-package repository contracts and runnable workflows.

## Scope
Applies under `tests/`.

## Local rules
- Keep tests deterministic, locally runnable, and self-cleaning.
- Unit tests remain the first proof for product behavior.

## Usage
Use child suites for command contracts, integration wiring, and smoke paths.

## Validation
Run each child suite through its Make target.

## Elements
| Element | Behavior |
| --- | --- |
| `commands` | Validates the canonical Make command interface. |
| `contracts` | Validates repository-owned external contracts. |
| `containers` | Validates Docker build targets and complete Compose wiring. |
| `smoke` | Exercises the complete local transaction-to-Kafka path and cleanup. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [commands/AGENTS.md](commands/AGENTS.md).
- Child: [contracts/AGENTS.md](contracts/AGENTS.md).
- Child: [containers/AGENTS.md](containers/AGENTS.md).
- Child: [smoke/AGENTS.md](smoke/AGENTS.md).
