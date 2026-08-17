# Infrastructure Documentation Instructions

## Purpose
Document cloud topology, local validation, and optional non-production verification.

## Scope
Applies to operator-facing infrastructure documentation in this directory.

## Local rules
- Never present an apply as part of automated validation.
- Require an explicitly selected non-production account for optional cloud commands.
- Keep variables, outputs, manifests, and Make targets synchronized.

## Usage
Read the reference before reviewing the optional AWS smoke procedure.

## Validation
Run `make test-infrastructure`, `make terraform-validate`, and `make k8s-validate`.

## Elements
| Element | Behavior |
| --- | --- |
| `aws-smoke.md` | Defines the optional, manually authorized non-production verification procedure. |
| `reference.md` | Explains prerequisites, variables, topology, configuration, and non-applying validation. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
