# DynamoDB Store Instructions

## Purpose
Provide DynamoDB transaction identity and outbox persistence.

## Scope
Applies to the DynamoDB adapter package.

## Local rules
- Use strongly consistent identity reads and conditional transactional writes.
- Keep partner/transaction identity and outbox records in one table contract.
- Classify conditional races as replay or conflict after rereading identity.
- Paginate filtered dispatch candidates and condition completion on the current claim token.
- Preserve exact producer object bytes as optional binary attributes with backward-safe reads.

## Usage
Select through `STORE_DRIVER=dynamodb` with IAM-provided access.

## Validation
Run AWS-client fake tests and the shared contract against DynamoDB Local.

## Elements
| Element | Behavior |
| --- | --- |
| `store.go` | Implements consistent identity/readiness checks, conditional race classification, paginated dispatch, and fenced outcomes. |
| `store_test.go` | Specifies consistent lookup, transactional races, pagination, fencing, replay, and conflict. |
| `table.go` | Idempotently bootstraps the single-table keys and pending-outbox GSI. |
| `table_test.go` | Specifies table creation, idempotency, readiness waiting, and failure classification. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
