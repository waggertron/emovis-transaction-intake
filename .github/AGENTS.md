# GitHub Instructions

## Purpose
Govern GitHub-hosted automation.

## Scope
Applies under `.github/`.

## Local rules
- Use least-privilege permissions, pinned major actions, and canonical Make targets.
- Do not store credentials or apply cloud infrastructure.

## Usage
Use workflows for repeatable validation of changes and the published repository.

## Validation
Run the CI contract test and reproduce every workflow command locally.

## Elements
| Element | Behavior |
| --- | --- |
| `workflows` | Defines hosted validation workflows. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [workflows/AGENTS.md](workflows/AGENTS.md).
