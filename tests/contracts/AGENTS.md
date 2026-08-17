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
| `openapi_semantic_test.go` | Parses the OpenAPI document and verifies operation, response, media-type, and transaction-schema semantics. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
