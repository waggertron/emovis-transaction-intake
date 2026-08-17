# Terraform Instructions

## Purpose
Define the non-production reference AWS topology for transaction intake.

## Scope
Applies to Terraform configuration in this directory.

## Local rules
- Pin provider versions and use no remote backend in the assignment reference.
- Use private subnets, encryption, TLS, IAM/IRSA, deletion protection controls, and explicit logs/alarms.
- Treat real AWS planning/apply as optional and account-selected; CI validates only.

## Usage
Copy `terraform.tfvars.example`, review costs, and use the Make targets; never apply implicitly.

## Validation
Run format check, backend-disabled initialization, validation, policy tests, and mock-input plan.

## Elements
| Element | Behavior |
| --- | --- |
| `.terraform.lock.hcl` | Locks checksummed provider selections for reproducible validation. |
| `versions.tf` | Pins Terraform and AWS provider versions and configures no-credential validation mode. |
| `variables.tf` | Declares topology, sizing, image, and safety inputs. |
| `main.tf` | Defines networking, EKS, MSK, DynamoDB, RDS, secrets, IAM, logs, and alarms. |
| `outputs.tf` | Exposes non-secret deployment integration values. |
| `terraform.tfvars.example` | Provides non-production, no-credential validation inputs. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
