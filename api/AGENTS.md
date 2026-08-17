# API Contract Instructions

## Purpose
Maintain the explicitly repository-owned mock OpenAPI contract.

## Scope
Applies to API description files under `api/`.

## Local rules
- Label the contract as mocked because no source specification was supplied.
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
