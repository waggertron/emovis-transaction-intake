# Topic Bootstrap Command Instructions

## Purpose
Configure the transaction review-candidate topic idempotently.

## Scope
Applies to the topic-bootstrap executable.

## Local rules
- Require explicit broker configuration and use bounded connection timeouts.
- Never log SASL credentials or connection strings containing secrets.
- Preserve tested partitions and retention defaults.

## Usage
Run through Docker Compose locally or the EKS bootstrap Job in cloud deployments.

## Validation
Run command tests and the Compose bootstrap smoke check.

## Elements
| Element | Behavior |
| --- | --- |
| `main.go` | Parses settings, connects to the controller, and ensures the topic. |
| `main_test.go` | Specifies topic-bootstrap environment parsing and defaults. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
