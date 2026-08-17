# Deployment Instructions

## Purpose
Contain hardened, reviewable workload manifests for the cloud topology.

## Scope
Applies under `deploy/`.

## Local rules
- Pin images by digest, run non-root, drop capabilities, use read-only filesystems, and set resource bounds.
- Reference externally managed secrets; never place secret values in manifests.
- Validate locally and never apply from CI.

## Usage
Render environment-specific names/images before an explicitly authorized deployment.

## Validation
Run `make k8s-validate` and infrastructure policy tests.

## Elements
| Element | Behavior |
| --- | --- |
| `kubernetes` | Defines API, worker, topic bootstrap, service account, service, and disruption policy manifests. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [kubernetes/AGENTS.md](kubernetes/AGENTS.md).
