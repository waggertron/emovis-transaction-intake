---
name: kafka-best-practices
description: "Design, implement, review, test, or operate Apache Kafka systems, including producers, consumers, topics, schemas, retries, outbox workflows, Docker Compose, and Amazon MSK. Use when a task involves Kafka reliability, delivery semantics, partitioning, consumer lag, security, MSK, or Kafka configuration."
---

# Kafka Best Practices

## Delivery semantics

- State the intended guarantee: at-most-once, at-least-once, or exactly-once. Do not claim end-to-end exactly-once unless every boundary supports it.
- Use an application outbox when a database write and Kafka publication must both occur. Persist the business record and pending outbox event atomically; publish asynchronously.
- Treat outbox publication as at-least-once. Include an immutable event ID and require consumers to deduplicate it.
- Use producer idempotence when supported. Keep compatible acknowledgements, retry, and in-flight-request settings; do not rely on it to replace consumer or business idempotency.
- Commit consumer offsets only after successful, idempotent processing. Make retries, poison-message handling, and terminal failure destinations explicit.

## Event and topic design

- Define an event envelope with `eventId`, `eventType`, `schemaVersion`, `occurredAt`, producer identity, correlation ID, and payload.
- Keep PII, credentials, raw payment data, and unnecessary request headers out of events and logs. Minimize payloads and document retention.
- Choose a key from the required ordering boundary. Use the same stable key for retries. Do not key all traffic by a high-volume tenant unless a hot partition is acceptable.
- Version events compatibly. Add optional fields instead of changing meanings; publish a new event type/version for breaking changes.
- Set partitions for expected parallelism, not arbitrary volume. Configure replication, `min.insync.replicas`, acknowledgements, cleanup policy, and retention as an explicit durability and cost decision.
- Use `delete` retention for immutable event streams. Use compaction only when the topic represents the latest state by key and consumers understand tombstones.

## Producers and consumers

- Bound request, delivery, and retry timeouts; propagate cancellation from the application context.
- Surface publish failures with event ID, topic, key hash, attempt count, and error class. Never log secrets or whole sensitive payloads.
- Make consumer handlers idempotent. Separate retryable infrastructure failures from invalid or permanently unprocessable events.
- Put retry delays and dead-letter behavior in the design. A dead-letter record must preserve the original event ID, reason, and safe diagnostic metadata.
- Track consumer lag, publish latency/error rate, retry count, outbox backlog/age, dead-letter count, broker health, partition skew, and storage capacity.

## Local and Amazon MSK operations

- Use Docker Compose with a single-node KRaft Kafka broker only for local development. Do not infer production durability from it.
- Keep local test publishers/consumers deterministic; add broker integration tests after unit tests, not instead of them.
- For MSK, use TLS for client-broker traffic and encryption at rest. Use least-privilege network and IAM policies.
- Store SASL/SCRAM credentials in AWS Secrets Manager. For MSK association, use an `AmazonMSK_`-prefixed secret encrypted with a customer-managed KMS key; never use the default Secrets Manager key for that secret.
- Configure broker logging and CloudWatch/Prometheus monitoring. Alert on consumer lag, disk utilization, under-replicated partitions, controller/broker health, and sustained CPU pressure.

## Test-first workflow

1. Write and run a failing test for event construction, key selection, duplicate handling, retry outcome, or configuration validation.
2. Implement the smallest code to pass it.
3. Add adapter contract tests with a deterministic Kafka fake.
4. Add Compose/MSK integration checks only after unit tests are green.

## Sources

- [Apache Kafka producer configuration](https://kafka.apache.org/40/configuration/producer-configs/)
- [Apache Kafka topic configuration](https://kafka.apache.org/40/configuration/topic-level-configs/)
- [Amazon MSK SASL/SCRAM setup](https://docs.aws.amazon.com/msk/latest/developerguide/msk-password-tutorial.html)
- [Amazon MSK encryption](https://docs.aws.amazon.com/msk/latest/developerguide/msk-encryption.html)
- [Amazon MSK monitoring](https://docs.aws.amazon.com/msk/latest/developerguide/monitoring.html)
