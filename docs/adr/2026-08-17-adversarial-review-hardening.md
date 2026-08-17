# Adversarial review hardening

## Problem Statement

The first whole-repository adversarial review found that ordinary tests and scanners had missed cross-process correctness and deployment-composition failures: expired outbox owners could overwrite reassigned work, DynamoDB filtering could starve later records, concurrent duplicate intake could surface expected races as availability errors, readiness did not inspect storage, topic bootstrap referenced an undeclared Kubernetes Secret, and the build contract did not verify the ARM64 architecture selected for EKS.

## Options Evaluated

### Option A: Retain leases and procedural deployment notes

| Pros | Cons |
| --- | --- |
| No port or deployment changes | Stale workers can corrupt state; candidate starvation remains; deployment correctness depends on undocumented timing and external objects |

### Option B: Fence claims and make every boundary executable

| Pros | Cons |
| --- | --- |
| Stale completion is rejected; DynamoDB progresses across pages; readiness is meaningful; secrets and architecture have explicit gates | Changes every storage adapter and requires schema migration, stronger component tests, and additional deployment configuration |

### Option C: Replace the outbox with direct Kafka publication

| Pros | Cons |
| --- | --- |
| Removes lease bookkeeping | Reintroduces the storage/Kafka dual-write failure and makes valid intake depend synchronously on Kafka |

## Decision

Choose Option B. At-least-once publication remains the intended guarantee, but each claim now carries an opaque token and only the current token can complete the event. DynamoDB query pagination must continue until the requested eligible batch is filled or candidates are exhausted. Expected conditional, uniqueness, and serialization races are classified at the storage boundary. API readiness checks the selected intake store with a timeout and does not depend on asynchronous Kafka.

Topic bootstrap uses the same bounded local/AWS secret-provider model as the application through IRSA. Terraform keeps secret values external and gates SCRAM association and the bootstrap Job on the explicit `runtime_secrets_ready` operator confirmation. Linux ARM64 cross-compilation is part of the local and CI contract because the reference EKS node group uses Graviton.

## Implementation Details

- `PendingEvent` and `PublishFailure` carry a claim token; all completion writes condition on it and return `ErrLeaseLost` for stale owners.
- PostgreSQL adds `claim_token` through an idempotent `ALTER TABLE`; DynamoDB records it during conditional lease acquisition; local stores mirror the same contract.
- DynamoDB follows `LastEvaluatedKey`, and conditional acceptance races reread the strongly consistent identity. PostgreSQL classifies unique and serialization SQL states after rollback.
- Each production store implements bounded readiness. Kubernetes `/readyz` therefore removes an unavailable API replica from service.
- Topic-bootstrap loads `LOCAL_SECRET_FILE` or `AWS_SECRET_ID`; Kubernetes no longer references an undeclared Secret.
- `make build-arm64` cross-compiles and inspects both production commands, and CI runs it.
- Parsed OpenAPI and rendered-Kubernetes tests validate semantic requirements and cross-resource references rather than relying only on text presence.
