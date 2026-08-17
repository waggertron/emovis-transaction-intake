# Hierarchy Skill Instructions

## Purpose
Maintain directory-instruction validation.

## Scope
Applies to this skill and its children.

## Local rules
- Add failing tests before validator behavior changes.
- Parse structure and links rather than arbitrary text.

## Usage
Use when directories or `AGENTS.md` files change.

## Validation
Run validator tests and then validate the repository.

## Elements
| Element | Behavior |
| --- | --- |
| `SKILL.md` | Defines the hierarchy policy. |
| `agents` | Contains skill metadata. |
| `scripts` | Contains validation code. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [agents/AGENTS.md](agents/AGENTS.md).
- Child: [scripts/AGENTS.md](scripts/AGENTS.md).
