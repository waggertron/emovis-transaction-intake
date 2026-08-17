# Hierarchy Script Instructions

## Purpose
Govern executable hierarchy validation.

## Scope
Applies to validator scripts.

## Local rules
- Add focused failing tests before script behavior.
- Resolve Markdown tables and reciprocal links.

## Usage
Run from repository root.

## Validation
Run unit tests, then the repository hierarchy check.

## Elements
| Element | Behavior |
| --- | --- |
| `validate_hierarchy.py` | Checks coverage, sections, elements, and links. |
| `validate_hierarchy_test.py` | Exercises missing files/sections, parsed element rows, and reciprocal links. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
