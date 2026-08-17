# Infrastructure Test Instructions

## Purpose
Validate AWS and Kubernetes deployment contracts without applying resources or requiring cloud credentials.

## Scope
Applies to infrastructure contract and policy tests.

## Local rules
- Never run `terraform apply` or `kubectl apply`.
- Assert encryption, least privilege, private networking, probes, resources, non-root execution, and external secret references.
- Keep no-credential validation deterministic and self-cleaning.

## Usage
Run through `make test-infrastructure`, `make terraform-validate`, and `make k8s-validate`.

## Validation
The suite fails for missing resources, plaintext secrets, mutable images, public data services, or absent workload hardening.

## Elements
| Element | Behavior |
| --- | --- |
| `infrastructure_test.sh` | Checks required Terraform resources, policies, and Kubernetes workload controls. |
| `validate_kubernetes.py` | Parses the locally rendered manifests and rejects malformed or incomplete Kubernetes objects. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
