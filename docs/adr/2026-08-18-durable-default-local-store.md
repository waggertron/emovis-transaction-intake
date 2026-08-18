# Durable Default Local Store

## Problem Statement

The ingest acknowledgment promises durable acceptance and retry-safe idempotency. The default Compose service currently falls back to memory, so a restart loses transactions and pending outbox events. Local development must demonstrate the durability claim without cloud credentials.

## Options Evaluated

### Option A: Memory

| Pros | Cons |
| --- | --- |
| Fast and dependency-free | Loses all accepted transactions and outbox state on restart |

### Option B: NDJSON

| Pros | Cons |
| --- | --- |
| Credential-free, durable across restart, already implements the storage port | Single-process only; unsuitable for horizontally scaled production |

### Option C: PostgreSQL

| Pros | Cons |
| --- | --- |
| Strong transactional semantics and production-shaped local flow | Adds a database dependency to the default developer startup |

### Option D: DynamoDB Local

| Pros | Cons |
| --- | --- |
| Exercises the AWS adapter locally | Heavier default stack and differs from managed-service operations |

## Decision

Choose Option B for the default Compose workflow. Memory remains an explicitly ephemeral focused-development option. PostgreSQL and DynamoDB Local remain fully tested selectable implementations, while production infrastructure still requires an explicit operator decision.

## Implementation Details

- Set the application service to `STORE_DRIVER=ndjson` and mount a repository-scoped named volume.
- Create the store directory with least privilege before the non-root API starts.
- Add restart/replay/outbox smoke evidence and targeted cleanup through Make.
- Do not claim NDJSON supports multi-process production deployment.

