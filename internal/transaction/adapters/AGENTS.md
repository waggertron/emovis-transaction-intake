# Transaction Adapter Instructions

## Purpose
Translate external technology contracts into transaction application ports.

## Scope
Applies under `internal/transaction/adapters/`.

## Local rules
- Keep business decisions in domain/application packages.
- Test mappings and failure classification before implementation.

## Usage
Select concrete adapters only in composition roots.

## Validation
Run focused adapter tests and shared contract suites.

## Elements
| Element | Behavior |
| --- | --- |
| `http` | Maps the OpenAPI-shaped HTTP contract to application calls. |
| `dynamodb` | Provides conditional transactional DynamoDB transaction/outbox storage. |
| `kafka` | Publishes versioned transaction review-candidate events. |
| `memory` | Provides concurrency-safe in-memory transaction and outbox storage. |
| `ndjson` | Provides single-process append-only local transaction and outbox storage. |
| `postgres` | Provides transactional PostgreSQL/RDS transaction and outbox storage. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [http/AGENTS.md](http/AGENTS.md).
- Child: [dynamodb/AGENTS.md](dynamodb/AGENTS.md).
- Child: [kafka/AGENTS.md](kafka/AGENTS.md).
- Child: [memory/AGENTS.md](memory/AGENTS.md).
- Child: [ndjson/AGENTS.md](ndjson/AGENTS.md).
- Child: [postgres/AGENTS.md](postgres/AGENTS.md).
