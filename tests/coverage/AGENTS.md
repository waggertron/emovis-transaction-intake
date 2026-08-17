# Coverage Test Instructions

## Purpose
Verify that coverage enforcement evaluates every production Go package independently.

## Scope
Applies under `tests/coverage/`.

## Local rules
- Reject missing, malformed, stale, or below-threshold package results.
- Include command packages and never substitute aggregate coverage for package coverage.

## Usage
Run through the canonical `make coverage` target or execute the checker tests while developing the gate.

## Validation
Run `bash tests/coverage/coverage_gate_test.sh`.

## Elements
| Element | Behavior |
| --- | --- |
| `check.sh` | Validates a complete per-package Go coverage report against a threshold. |
| `coverage_gate_test.sh` | Exercises passing, below-threshold, missing-package, malformed, stale, and false-aggregate cases. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
