# API Contract Instructions

## Purpose
Maintain the supplied OpenAPI 3.0.3 ingest contract and documented operational extensions.

## Scope
Applies to API description files under `api/`.

## Local rules
- Treat the supplied ingest contract as authoritative; keep operational endpoints explicitly identified as repository extensions.
- Keep examples, status codes, headers, and schemas aligned with handler tests.
- Document method and media-type rejection and validate the document through a parsed semantic contract.

## Usage
Review before modifying HTTP requests or responses.

## Validation
Run the API contract test and HTTP adapter suite.

## Elements
| Element | Behavior |
| --- | --- |
| `openapi.yaml` | Defines transaction intake and operational HTTP endpoints. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
