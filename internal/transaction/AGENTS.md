# Transaction Instructions

## Purpose
Own transaction intake behavior and its ports and adapters.

## Scope
Applies under `internal/transaction/`.

## Local rules
- Use tolling-domain vocabulary and keep infrastructure out of the core.
- Preserve idempotent intake and atomic transaction/outbox semantics.

## Usage
Put pure invariants in `domain`, orchestration in application code, and technology mappings in adapters.

## Validation
Run package tests and every adapter's shared contract suite.

## Elements
| Element | Behavior |
| --- | --- |
| `adapters` | Translates storage, HTTP, and Kafka technologies at the core boundary. |
| `app` | Orchestrates intake and owns outbound port contracts. |
| `domain` | Defines transaction values, invariants, and deterministic fingerprints. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [adapters/AGENTS.md](adapters/AGENTS.md).
- Child: [app/AGENTS.md](app/AGENTS.md).
- Child: [domain/AGENTS.md](domain/AGENTS.md).
