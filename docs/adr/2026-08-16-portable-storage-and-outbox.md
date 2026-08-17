# Portable Storage and Transactional Outbox

## Problem Statement

Transaction intake must be idempotent and must not lose the review event when storage succeeds but Kafka is unavailable. The assignment also asks for local, DynamoDB, and PostgreSQL storage choices without coupling business behavior to one vendor.

## Options Evaluated

### Option A: Write storage, then publish synchronously

| Pros | Cons |
| --- | --- |
| Simple request flow | A crash between writes loses the event; Kafka latency affects intake |

### Option B: Cohesive storage port with transactional outbox

| Pros | Cons |
| --- | --- |
| Atomic transaction/event intent; Kafka outages do not reject valid intake; adapters share one contract | Requires a dispatcher, retry state, leases, and consumer deduplication |

### Option C: Kafka-first intake

| Pros | Cons |
| --- | --- |
| High ingestion throughput | Moves idempotency and durable query state to consumers; acceptance semantics become less direct |

## Decision

Choose Option B. The application owns a cohesive `TransactionStore` conversation, while each adapter atomically accepts the transaction and creates its pending event. Each deployment selects exactly one adapter; there are no dual writes, replication, or migrations between adapters in this assignment.

## Implementation Details

- `internal/transaction/app` owns intake and outbox ports.
- Memory and append-only NDJSON adapters implement local behavior; NDJSON is restricted to one combined process.
- The dispatcher claims leased batches, publishes asynchronously, and records success or bounded retry state.
- PostgreSQL uses one SQL transaction; DynamoDB uses one transactional write when their adapters are selected.
- Event IDs are stable across retries, and consumers must deduplicate them.
