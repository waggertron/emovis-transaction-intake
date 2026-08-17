# Testing Documentation Instructions

## Purpose
Map critical behavior to unit, component, and end-to-end evidence.

## Scope
Applies to testing strategy and evidence documents in this directory.

## Local rules
- Reference actual test or Make target names.
- Do not use numerical coverage as a substitute for behavioral assertions.

## Usage
Reconcile the matrix whenever behavior or test names change.

## Validation
Run `make validate-static`, named component gates, and named E2E gates.

## Elements
| Element | Behavior |
| --- | --- |
| `behavior-matrix.md` | Maps critical risks and behavior to explicit assertions at each applicable layer. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
