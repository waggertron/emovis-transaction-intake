# Contract Test Instructions

## Purpose
Validate repository-owned external API and event contracts.

## Scope
Applies to static contract checks.

## Local rules
- Cross-check the supplied OpenAPI baseline byte-for-byte and semantically against executable adapter tests.

## Usage
Run after any OpenAPI or event-schema change.

## Validation
Run `bash tests/contracts/openapi_test.sh`.

## Elements
| Element | Behavior |
| --- | --- |
| `openapi_test.sh` | Verifies the API file remains identical to the supplied baseline. |
| `openapi_semantic_test.go` | Parses the OpenAPI document and verifies supplied prose, operation, response, media-type, named-schema, and transaction semantics. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
