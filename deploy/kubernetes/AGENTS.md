# Kubernetes Deployment Instructions

## Purpose
Define separate production API, worker, and post-MSK topic-bootstrap workloads.

## Scope
Applies to Kubernetes manifests in this directory.

## Local rules
- Keep API and worker independently scalable and use one IRSA-enabled service account.
- Require probes where a network health surface exists and resource/security bounds everywhere.
- Load application and topic-bootstrap credentials directly from AWS Secrets Manager through IRSA.

## Usage
Replace example immutable image digests and environment endpoints during an authorized deployment.

## Validation
Run client-side dry-run/schema checks and repository infrastructure policy tests.

## Elements
| Element | Behavior |
| --- | --- |
| `namespace.yaml` | Declares the namespace required by every namespaced workload and policy. |
| `service-account.yaml` | Declares the IRSA-annotated workload identity placeholder. |
| `api.yaml` | Runs the horizontally scalable HTTP intake deployment and service. |
| `api-hpa.yaml` | Bounds CPU-driven scaling for the API deployment. |
| `disruption-budgets.yaml` | Preserves API and worker availability during voluntary disruptions. |
| `worker.yaml` | Runs the independently scalable outbox dispatcher. |
| `worker-hpa.yaml` | Bounds CPU-driven scaling for the worker deployment. |
| `topic-bootstrap-job.yaml` | Loads bounded AWS configuration and runs idempotent topic configuration after MSK readiness. |
| `kustomization.yaml` | Provides a deterministic local render without consulting a cluster. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
