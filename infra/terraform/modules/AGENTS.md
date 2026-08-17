# Terraform Module Instructions

## Purpose
Contain explicit, independently selectable infrastructure responsibilities.

## Scope
Applies to all Terraform child modules.

## Local rules
- Keep module inputs explicit and outputs minimal.
- Do not introduce a default persistence selection.

## Usage
Instantiate modules only from the root Terraform configuration.

## Validation
Run formatting, validation, infrastructure contracts, and both example plans.

## Elements
| Element | Behavior |
| --- | --- |
| `shared` | Owns storage-independent secret containers. |
| `dynamodb` | Owns the optional DynamoDB store. |
| `postgres` | Owns the optional PostgreSQL store. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [shared/AGENTS.md](shared/AGENTS.md).
- Child: [dynamodb/AGENTS.md](dynamodb/AGENTS.md).
- Child: [postgres/AGENTS.md](postgres/AGENTS.md).
