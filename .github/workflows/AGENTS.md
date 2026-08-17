# Workflow Instructions

## Purpose
Define hosted validation for code, contracts, containers, and smoke behavior.

## Scope
Applies to workflow YAML files.

## Local rules
- Keep permissions read-only and route validation through Make.
- Never run Terraform/Kubernetes apply or require AWS credentials.

## Usage
Run on pushes and pull requests to protect the main branch.

## Validation
Run `bash tests/ci/workflow_test.sh` and all referenced Make targets.

## Elements
| Element | Behavior |
| --- | --- |
| `ci.yml` | Runs static, component, end-to-end, infrastructure, and smoke gates without cloud apply. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
