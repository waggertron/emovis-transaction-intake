# DynamoDB Store Instructions

## Purpose
Provide DynamoDB transaction identity and outbox persistence.

## Scope
Applies to the DynamoDB adapter package.

## Local rules
- Use strongly consistent identity reads and conditional transactional writes.
- Keep partner/transaction identity and outbox records in one table contract.
- Classify conditional races as replay or conflict after rereading identity.

## Usage
Select through `STORE_DRIVER=dynamodb` with IAM-provided access.

## Validation
Run AWS-client fake tests and the shared contract against DynamoDB Local.

## Elements
| Element | Behavior |
| --- | --- |
| `store.go` | Implements consistent identity reads and conditional transaction/outbox writes. |
| `store_test.go` | Specifies consistent lookup, transactional writes, replay, and conflict. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
