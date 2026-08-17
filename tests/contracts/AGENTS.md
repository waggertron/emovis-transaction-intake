# Contract Test Instructions

## Purpose
Validate repository-owned external API and event contracts.

## Scope
Applies to static contract checks.

## Local rules
- Cross-check contract content against executable adapter tests.

## Usage
Run after any OpenAPI or event-schema change.

## Validation
Run `bash tests/contracts/openapi_test.sh`.

## Elements
| Element | Behavior |
| --- | --- |
| `openapi_test.sh` | Verifies required API paths, fields, responses, and headers. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
