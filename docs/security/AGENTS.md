# Security Documentation Instructions

## Purpose
Preserve evidence-based release security review results.

## Scope
Applies to security reports in this directory.

## Local rules
- Number findings and distinguish fixed, accepted, and false-positive states.
- Include reproducible commands without credentials or destructive cloud operations.
- Do not claim a scanner covered policies it skipped.

## Usage
Read the latest report before release or architecture changes.

## Validation
Run `make security`, `make test-race`, and the affected infrastructure/product gates.

## Elements
| Element | Behavior |
| --- | --- |
| `2026-08-17-final-review.md` | Records scope, findings, remediation, limitations, and release evidence. |
| `2026-08-17-adversarial-follow-up.md` | Reopens the release assessment and tracks remediation of cross-cutting adversarial findings. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
