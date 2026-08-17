# Kafka and Amazon MSK Delivery

## Problem Statement

Accepted transactions need asynchronous review-event delivery with ordering by partner transaction, bounded retries, observable lag, local reproducibility, and an AWS production shape.

## Options Evaluated

### Option A: SNS and SQS

| Pros | Cons |
| --- | --- |
| Native AWS operations and simple fan-out | Does not align with the accepted Kafka requirement or local broker contract |

### Option B: Apache Kafka locally and Amazon MSK in AWS

| Pros | Cons |
| --- | --- |
| Stable keyed ordering, replayable log, local KRaft broker, managed AWS option | More operational and security configuration than queues |

### Option C: In-process dispatch only

| Pros | Cons |
| --- | --- |
| Minimal infrastructure | No durable cross-process delivery, scaling, or replay |

## Decision

Choose Option B with `kafka-go`. Delivery is explicitly at least once, keyed by `partnerId:transactionId`, and uses a stable outbox `eventId`. End-to-end exactly once is not claimed.

## Implementation Details

- `transaction.review-candidates.v1` defaults to three partitions and seven-day delete retention.
- `topic-bootstrap` applies configuration idempotently after broker readiness.
- The producer requires all acknowledgements, stable hash partitioning, bounded timeouts, and five attempts.
- Local Compose uses one KRaft broker. MSK uses TLS and SASL/SCRAM credentials supplied by Secrets Manager.
- Consumers deduplicate `eventId`; operations monitor publish failures, outbox age/backlog, and consumer lag.
