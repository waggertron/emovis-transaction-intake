# Test Fixture Instructions

## Purpose
Contain deterministic fixtures used only by local component and E2E services.

## Scope
Applies under `tests/fixtures/`.

## Local rules
- Never reuse fixture credentials or certificates outside isolated local tests.
- Generate private keys and mutable credentials at runtime and clean them with the owning Compose project.

## Usage
Mount only through named test profiles.

## Validation
Run the consuming component and E2E targets and verify teardown removes generated state.

## Elements
| Element | Behavior |
| --- | --- |
| `kafka-secure` | Bootstraps isolated TLS/SCRAM Kafka test state. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [kafka-secure/AGENTS.md](kafka-secure/AGENTS.md).
