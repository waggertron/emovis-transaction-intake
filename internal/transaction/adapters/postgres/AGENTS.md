# PostgreSQL Store Instructions

## Purpose
Provide RDS PostgreSQL transaction and outbox persistence.

## Scope
Applies to the PostgreSQL adapter package.

## Local rules
- Use `database/sql` with pgx and parameterized statements only.
- Accept transaction and outbox intent in one SQL transaction.
- Use row locks and `SKIP LOCKED` leases for concurrent dispatchers.
- Fence completion by claim token and classify uniqueness/serialization races after rollback.
- Preserve exact producer object bytes in separate `BYTEA` columns while retaining structured JSONB payloads.

## Usage
Select through `STORE_DRIVER=postgres` with credentials supplied externally.

## Validation
Run sqlmock unit tests, the shared store contract, and Compose PostgreSQL integration tests.

## Elements
| Element | Behavior |
| --- | --- |
| `migrate.go` | Embeds and idempotently applies the PostgreSQL schema at component or process bootstrap. |
| `schema.sql` | Defines transaction identity, raw audit columns, idempotently migrated claim tokens, and leased outbox indexes. |
| `store.go` | Implements parameterized atomic acceptance, raw audit mapping, readiness, concurrency classification, and fenced outcomes. |
| `store_test.go` | Specifies SQL transactions, readiness, race classification, fencing, idempotency, and rollback. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
