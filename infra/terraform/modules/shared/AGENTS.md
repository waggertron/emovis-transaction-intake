# Shared Terraform Module Instructions

## Purpose
Own storage-independent runtime secret containers.

## Scope
Applies to the shared Terraform module.

## Local rules
- Keep this module independent of the selected persistence backend.
- Create secret containers only; values remain externally populated.

## Usage
The root module instantiates this module once for every explicit deployment plan.

## Validation
Run `make terraform-validate` and both explicitly selected example plans.

## Elements
| Element | Behavior |
| --- | --- |
| `main.tf` | Creates API and Kafka secret containers encrypted by the supplied KMS key. |
| `variables.tf` | Declares naming, encryption, and tag inputs. |
| `outputs.tf` | Exposes non-secret names and ARNs for root wiring. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
