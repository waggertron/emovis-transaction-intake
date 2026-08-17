# DynamoDB Terraform Module Instructions

## Purpose
Own the explicitly selected DynamoDB transaction and outbox store.

## Scope
Applies to the DynamoDB Terraform module.

## Local rules
- Preserve the application key and outbox-dispatch index contract.
- Retain encryption, point-in-time recovery, and deletion protection controls.

## Usage
The root module instantiates this module only for `storage_backend="dynamodb"`.

## Validation
Run infrastructure contracts and the DynamoDB example plan.

## Elements
| Element | Behavior |
| --- | --- |
| `main.tf` | Creates the encrypted transaction/outbox table and dispatch index. |
| `variables.tf` | Declares naming, encryption, protection, and tag inputs. |
| `outputs.tf` | Exposes table identity for runtime configuration and least-privilege IAM. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
