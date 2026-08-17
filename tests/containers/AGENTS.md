# Container Contract Instructions

## Purpose
Validate reproducible image targets and complete local service wiring.

## Scope
Applies to static Dockerfile and Compose contract tests.

## Local rules
- Do not start containers in static tests.
- Require non-root runtime images, health checks, explicit dependencies, and pinned images.
- Require a locally and CI-validated Linux ARM64 artifact path for the selected Graviton nodes.

## Usage
Run before implementing or modifying container definitions.

## Validation
Run `bash tests/containers/definitions_test.sh`, then `make compose-config`.

## Elements
| Element | Behavior |
| --- | --- |
| `definitions_test.sh` | Checks Docker targets, ARM64 build evidence, security posture, and Compose services. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
