# Command Contract Test Instructions

## Purpose
Validate the strict Makefile interface exposed to developers and CI.

## Scope
Applies to command-contract tests.

## Local rules
- Test target discovery and safe defaults without starting services.
- Fail with actionable missing-target messages.

## Usage
Run before implementing or changing Make targets.

## Validation
Run `bash tests/commands/makefile_test.sh`.

## Elements
| Element | Behavior |
| --- | --- |
| `makefile_test.sh` | Verifies required strict targets and help output. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
