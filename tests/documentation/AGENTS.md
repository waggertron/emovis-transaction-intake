# Documentation Test Instructions

## Purpose
Validate repository-relative Markdown links without network access.

## Scope
Applies to documentation validation scripts in this directory.

## Local rules
- Ignore external URLs and anchors; resolve local links from their source file.
- Exclude generated and dependency directories consistently with hierarchy validation.

## Usage
Run through `make docs-validate`.

## Validation
Run the checker against the repository root.

## Elements
| Element | Behavior |
| --- | --- |
| `link_check.py` | Rejects missing repository-relative Markdown link targets. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
