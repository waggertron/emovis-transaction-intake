# Transaction and Kafka Flow

This service accepts toll transactions over HTTP, stores them safely, and asynchronously publishes accepted transactions to Kafka for downstream review. Kafka is an outbound event stream; it is not the primary transaction database and does not participate directly in the HTTP request.

## Request-to-Kafka flow

```text
roadside or partner system
          │ POST /ingest/v1/transactions
          ▼
      HTTP adapter
          │ parse, authenticate, and validate
          ▼
    intake application
          │ atomically accept
          ├── transaction record
          └── pending outbox event
                         │ background worker claims event
                         ▼
                   Kafka publisher
                         │
                         ▼
             transaction.review-candidates.v1
                         │
                         ▼
              downstream review systems
```

### 1. HTTP intake

A client sends JSON to `POST /ingest/v1/transactions`. The [HTTP adapter](../../internal/transaction/adapters/http/handler.go) limits the request body, requires the JSON media type, rejects unknown fields, parses the transaction timestamp, and optionally authenticates an API key. It translates the request into the transport-independent transaction type used by the application.

### 2. Validation and idempotency

The [domain transaction](../../internal/transaction/domain/transaction.go) validates the source identifiers, transaction type, timestamp, decimal amount, currency, plate, and transponder fields. It also produces a canonical SHA-256 fingerprint of the submitted content.

The logical idempotency identity is `source:source_reference`:

- A new identity is accepted and receives a system transaction ID.
- The same identity and fingerprint is an idempotent replay. The service returns the original transaction ID and creates no second event.
- The same identity with a different fingerprint is a conflict and cannot silently overwrite the first billable record.

### 3. Atomic transaction and event creation

The [intake service](../../internal/transaction/app/intake.go) constructs the transaction record and a `transaction.review-candidate` outbox event. The event has an immutable event ID, schema version, UTC occurrence time, request correlation ID, transaction ID, Kafka key, and validated payload.

The selected store saves the transaction and pending event as one atomic operation. PostgreSQL does this in a serializable SQL transaction; DynamoDB uses a transactional write; the local memory and NDJSON implementations provide the equivalent application contract. Either both records are accepted or neither is.

### 4. Outbox dispatch

The API does not publish directly to Kafka. A separate [outbox dispatcher](../../internal/transaction/app/outbox.go) claims pending events in bounded batches using time-limited leases and opaque claim tokens. The worker publishes each claimed event, then marks it as published only if it still owns the claim.

If publication fails, the event remains in storage. Retries use bounded exponential delays and become terminal after the configured maximum number of attempts. Claim fencing prevents a slow worker whose lease expired from overwriting the result of a newer worker.

### 5. Kafka publication

The [Kafka adapter](../../internal/transaction/adapters/kafka/publisher.go) serializes the outbox event into a versioned JSON envelope and sends it to `transaction.review-candidates.v1`. The envelope contains:

- `eventId`, `eventType`, and `schemaVersion`
- occurrence and correlation timestamps/identifiers
- source, source reference, and system transaction ID
- the validated toll transaction payload

API keys, authorization headers, Kafka credentials, and payment credentials are not included.

The message key is `source:source_reference`. The [Kafka writer](../../internal/transaction/adapters/kafka/writer.go) uses hash partitioning, so events for the same source transaction are routed to the same partition and retain their relative order. It uses synchronous publication, acknowledgements from all required replicas, bounded timeouts, and producer retries. Automatic topic creation is disabled.

## Why Kafka is used

Kafka decouples durable transaction intake from downstream review, matching, billing, or settlement work. The API can accept a valid transaction without waiting for those systems to finish and without making their availability part of API readiness.

Kafka provides several useful properties:

- **Asynchronous processing:** review happens outside the request latency budget.
- **Buffering:** accepted events can wait while consumers are temporarily slow or unavailable.
- **Horizontal scaling:** consumer-group members can divide topic partitions among instances.
- **Per-key ordering:** related events remain ordered within their partition.
- **Replay:** retained events can be consumed again to rebuild downstream state.
- **Service decoupling:** intake does not need to know how many downstream systems process the event.

This repository contains the producer side only. A downstream resolution pipeline is expected to consume the topic elsewhere.

## Why the transactional outbox is necessary

Saving a transaction and then publishing directly creates a failure window:

```text
1. Store the transaction successfully.
2. Crash before publishing to Kafka.
3. The accepted transaction never reaches downstream review.
```

The outbox removes that lost-event window by saving the transaction and publication intent together:

```text
Atomic store operation:
    save transaction
    save pending outbox event

Asynchronous worker:
    claim pending event
    publish to Kafka
    mark event published
```

A Kafka outage therefore does not erase an accepted transaction or its pending review event. It delays delivery while the event remains available for retry.

## Delivery guarantee

Delivery is explicitly **at-least-once**, not exactly once. A duplicate is possible if Kafka accepts a message and the worker crashes before marking its outbox record as published. After the lease expires, another worker can publish that same event again.

The event keeps the same `eventId` across every retry. Downstream consumers must be idempotent and deduplicate using that immutable ID. Kafka producer retries improve reliability, but they do not replace consumer-side deduplication.

## Topic and security configuration

The [topic bootstrap](../../internal/transaction/adapters/kafka/topic.go) creates the topic if needed and treats an existing topic as success. The local defaults are three partitions and seven-day delete retention.

Local Docker Compose runs a single-node plaintext KRaft broker for reproducible development and tests. The production-oriented Amazon MSK configuration uses TLS and optional SASL/SCRAM-SHA-512 credentials loaded from external secret providers. Local single-node behavior is not evidence of production durability.

## Runtime modes

- `api` runs HTTP intake and writes to a shared selected store.
- `worker` reads that store's outbox and publishes to Kafka.
- `local` runs API and worker together against one shared memory or NDJSON store.
- `topic-bootstrap` creates the Kafka topic before producers start.

The API readiness check depends on the selected transaction store, not Kafka. In production, API and worker instances can scale independently while sharing PostgreSQL or DynamoDB.

## Verification

The local smoke test exercises HTTP intake, idempotent replay, the outbox, real Kafka publication, message-key selection, safe event content, and cleanup. Run it with:

```bash
make smoke
```

Focused and secure Kafka paths are available through `make test-component-kafka` and `make test-component-kafka-secure`. The complete behavior evidence is indexed in the [behavior-to-test matrix](../testing/behavior-matrix.md).
