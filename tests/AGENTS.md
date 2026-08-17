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
| `ci` | Validates hosted CI permissions and canonical command usage. |
| `component` | Runs shared behavioral contracts against local external-service implementations. |
| `contracts` | Validates repository-owned external contracts. |
| `containers` | Validates Docker build targets and complete Compose wiring. |
| `coverage` | Verifies per-package production coverage enforcement. |
| `documentation` | Validates repository-relative Markdown links. |
| `e2e` | Exercises every local implementation through production process boundaries. |
| `fixtures` | Contains deterministic, non-production local service bootstrap fixtures. |
| `infrastructure` | Validates Terraform and Kubernetes cloud-equivalence contracts locally. |
| `smoke` | Exercises the complete local transaction-to-Kafka path and cleanup. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [commands/AGENTS.md](commands/AGENTS.md).
- Child: [ci/AGENTS.md](ci/AGENTS.md).
- Child: [component/AGENTS.md](component/AGENTS.md).
- Child: [contracts/AGENTS.md](contracts/AGENTS.md).
- Child: [containers/AGENTS.md](containers/AGENTS.md).
- Child: [coverage/AGENTS.md](coverage/AGENTS.md).
- Child: [documentation/AGENTS.md](documentation/AGENTS.md).
- Child: [e2e/AGENTS.md](e2e/AGENTS.md).
- Child: [fixtures/AGENTS.md](fixtures/AGENTS.md).
- Child: [infrastructure/AGENTS.md](infrastructure/AGENTS.md).
- Child: [smoke/AGENTS.md](smoke/AGENTS.md).
