# Infrastructure Instructions

## Purpose
Define reproducible cloud infrastructure without embedding credentials or applying from CI.

## Scope
Applies under `infra/`.

## Local rules
- Encrypt data services, keep control/data planes private, and grant least privilege.
- Support no-credential formatting, initialization, validation, and example planning.
- Never add an automatic apply or destroy workflow.

## Usage
Use only the canonical Terraform Make targets.

## Validation
Run `make test-infrastructure`, `make terraform-fmt`, and `make terraform-validate`.

## Elements
| Element | Behavior |
| --- | --- |
| `terraform` | Provisions the AWS network, EKS, MSK, storage, IAM, secrets, logging, and alarms. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [terraform/AGENTS.md](terraform/AGENTS.md).
