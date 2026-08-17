# PostgreSQL Store Instructions

## Purpose
Provide RDS PostgreSQL transaction and outbox persistence.

## Scope
Applies to the PostgreSQL adapter package.

## Local rules
- Use `database/sql` with pgx and parameterized statements only.
- Accept transaction and outbox intent in one SQL transaction.
- Use row locks and `SKIP LOCKED` leases for concurrent dispatchers.

## Usage
Select through `STORE_DRIVER=postgres` with credentials supplied externally.

## Validation
Run sqlmock unit tests, the shared store contract, and Compose PostgreSQL integration tests.

## Elements
| Element | Behavior |
| --- | --- |
| `migrate.go` | Embeds and idempotently applies the PostgreSQL schema at component or process bootstrap. |
| `schema.sql` | Defines transaction identity and leased outbox tables and indexes. |
| `store.go` | Implements parameterized atomic PostgreSQL acceptance. |
| `store_test.go` | Specifies SQL transaction, idempotency, and rollback behavior. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
