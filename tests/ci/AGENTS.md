# CI Contract Instructions

## Purpose
Validate hosted CI permissions and use of canonical Make targets.

## Scope
Applies to static CI workflow checks.

## Local rules
- Require least-privilege permissions and no cloud credentials or infrastructure apply.
- Route product validation through Make targets.

## Usage
Run before adding or changing CI workflows.

## Validation
Run `bash tests/ci/workflow_test.sh`.

## Elements
| Element | Behavior |
| --- | --- |
| `workflow_test.sh` | Verifies CI triggers, action pins, permissions, and gates. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
