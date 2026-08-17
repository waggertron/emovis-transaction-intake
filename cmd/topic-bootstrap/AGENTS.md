# Topic Bootstrap Command Instructions

## Purpose
Configure the transaction review-candidate topic idempotently.

## Scope
Applies to the topic-bootstrap executable.

## Local rules
- Require explicit broker configuration and use bounded connection timeouts.
- Never log SASL credentials or connection strings containing secrets.
- Preserve tested partitions and retention defaults.
- Load bounded local or AWS Secrets Manager configuration directly; do not depend on undeclared Kubernetes Secrets.

## Usage
Run through Docker Compose locally or the EKS bootstrap Job in cloud deployments.

## Validation
Run command tests and the Compose bootstrap smoke check.

## Elements
| Element | Behavior |
| --- | --- |
| `main.go` | Loads external configuration, connects to the controller, and ensures the topic. |
| `main_test.go` | Specifies provider selection, environment parsing, defaults, and failures. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
