# PostgreSQL Terraform Module Instructions

## Purpose
Own the explicitly selected RDS PostgreSQL transaction and outbox store.

## Scope
Applies to the PostgreSQL Terraform module.

## Local rules
- Keep RDS private, encrypted, multi-AZ, backed up, and deletion-protected by default.
- Create the PostgreSQL secret container without populating application credentials.

## Usage
The root module instantiates this module only for `storage_backend="postgres"`.

## Validation
Run infrastructure contracts and the PostgreSQL example plan.

## Elements
| Element | Behavior |
| --- | --- |
| `main.tf` | Creates the subnet group, secret container, and hardened RDS instance. |
| `variables.tf` | Declares network, sizing, encryption, protection, and tag inputs. |
| `outputs.tf` | Exposes the sensitive endpoint and secret ARN for root wiring. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
