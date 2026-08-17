# Internal Code Instructions

## Purpose
Contain application code that is not a public Go library.

## Scope
Applies under `internal/`.

## Local rules
- Keep dependencies pointing from adapters toward application and domain code.
- Create focused failing tests before production behavior.

## Usage
Choose the business-area child package before architecture-oriented grouping.

## Validation
Run focused tests, then `go test ./...` and the race detector.

## Elements
| Element | Behavior |
| --- | --- |
| `bootstrap` | Loads configuration and constructs secure process boundaries. |
| `transaction` | Owns transaction intake domain and application behavior. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [bootstrap/AGENTS.md](bootstrap/AGENTS.md).
- Child: [transaction/AGENTS.md](transaction/AGENTS.md).
