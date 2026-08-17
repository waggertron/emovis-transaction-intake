# Production Service Bootstrap Metadata Instructions

## Purpose
Govern production-service-bootstrap skill metadata.

## Scope
Applies to this metadata directory.

## Local rules
- Keep discovery text aligned with the parent skill and secret-free.

## Usage
Update when the skill trigger or default prompt changes.

## Validation
Run the skill validator.

## Elements
| Element | Behavior |
| --- | --- |
| `openai.yaml` | Declares display metadata and an invocation prompt. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
