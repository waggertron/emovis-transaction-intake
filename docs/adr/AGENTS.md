# Architecture Decision Record Instructions

## Purpose
Preserve significant technical decisions and their rationale.

## Scope
Applies to ADRs under `docs/adr/`.

## Local rules
- Use dated filenames and include problem, options, decision, and implementation sections.
- Record trade-offs honestly and supersede rather than silently rewrite decisions.

## Usage
Read relevant ADRs before changing storage, delivery, deployment, or dependency direction.

## Validation
Check required sections, links, and consistency with the active plan and README.

## Elements
| Element | Behavior |
| --- | --- |
| `2026-08-16-portable-storage-and-outbox.md` | Chooses ports/adapters with atomic storage-owned outbox state. |
| `2026-08-16-kafka-msk-delivery.md` | Chooses Kafka/MSK at-least-once review-event delivery. |
| `2026-08-17-adversarial-review-hardening.md` | Chooses fenced outbox ownership and executable readiness, secret, contract, and architecture boundaries. |
| `2026-08-17-explicit-storage-infrastructure-selection.md` | Requires an operator-selected DynamoDB or PostgreSQL Terraform module with no default backend. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
