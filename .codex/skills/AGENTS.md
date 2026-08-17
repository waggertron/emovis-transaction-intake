# Skill Instructions

## Purpose
Govern repository-local skills.

## Scope
Applies to `.codex/skills/`.

## Local rules
- Keep each skill actionable and consistent with the current plan.

## Usage
Read the matching child skill before work in its domain.

## Validation
Run the skill validator and hierarchy validator.

## Elements
| Element | Behavior |
| --- | --- |
| `agent-instruction-hierarchy` | Validates directory instructions. |
| `bootstrap-production-service` | Establishes a complete production-service repository workflow. |
| `ci-portability` | Prevents environment assumptions from diverging between local and hosted validation. |
| `coverage-and-behavior` | Combines per-package coverage floors with critical-behavior evidence. |
| `engineering-choices` | Constrains technology choices. |
| `kafka-best-practices` | Defines Kafka reliability and security. |
| `local-cloud-equivalence` | Requires production adapters to pass against documented local substitutes. |
| `record-plan-decisions` | Records attributed material decisions in the active plan. |
| `secure-component-e2e` | Defines secure real-service component and end-to-end test contracts. |
| `tdd-development` | Enforces red-green-refactor. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [agent-instruction-hierarchy/AGENTS.md](agent-instruction-hierarchy/AGENTS.md).
- Child: [bootstrap-production-service/AGENTS.md](bootstrap-production-service/AGENTS.md).
- Child: [ci-portability/AGENTS.md](ci-portability/AGENTS.md).
- Child: [coverage-and-behavior/AGENTS.md](coverage-and-behavior/AGENTS.md).
- Child: [engineering-choices/AGENTS.md](engineering-choices/AGENTS.md).
- Child: [kafka-best-practices/AGENTS.md](kafka-best-practices/AGENTS.md).
- Child: [local-cloud-equivalence/AGENTS.md](local-cloud-equivalence/AGENTS.md).
- Child: [record-plan-decisions/AGENTS.md](record-plan-decisions/AGENTS.md).
- Child: [secure-component-e2e/AGENTS.md](secure-component-e2e/AGENTS.md).
- Child: [tdd-development/AGENTS.md](tdd-development/AGENTS.md).
