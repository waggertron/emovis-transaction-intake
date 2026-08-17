# Terraform Instructions

## Purpose
Define the non-production reference AWS topology for transaction intake.

## Scope
Applies to Terraform configuration in this directory.

## Local rules
- Pin provider versions and use no remote backend in the assignment reference.
- Use private subnets, encryption, TLS, IAM/IRSA, deletion protection controls, and explicit logs/alarms.
- Treat real AWS planning/apply as optional and account-selected; CI validates only.
- Gate SCRAM association and topic bootstrap on explicit external secret-population confirmation.

## Usage
Choose one backend-specific example file, review costs, and use the Make targets; never apply implicitly.

## Validation
Run format check, backend-disabled initialization, validation, policy tests, and mock-input plan.

## Elements
| Element | Behavior |
| --- | --- |
| `.terraform.lock.hcl` | Locks checksummed provider selections for reproducible validation. |
| `versions.tf` | Pins Terraform and AWS provider versions and configures no-credential validation mode. |
| `variables.tf` | Declares topology, required storage selection, sizing, image, secret-readiness, and safety inputs. |
| `main.tf` | Defines shared networking, EKS, MSK, selectable storage modules, gated bootstrap, IAM, logs, and alarms. |
| `outputs.tf` | Exposes non-secret deployment integration values. |
| `dynamodb.tfvars.example` | Explicitly selects DynamoDB for a non-production, no-credential example plan. |
| `postgres.tfvars.example` | Explicitly selects PostgreSQL for a non-production, no-credential example plan. |
| `modules` | Separates shared, DynamoDB, and PostgreSQL resource ownership. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [modules/AGENTS.md](modules/AGENTS.md).
