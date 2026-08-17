# Kafka Adapter Instructions

## Purpose
Publish transaction review candidates through Kafka.

## Scope
Applies to the Kafka adapter package.

## Local rules
- Preserve stable event IDs, keys, schema versions, and at-least-once semantics.
- Never include credentials, API keys, or unnecessary PII in messages or errors.
- Test message mapping with a deterministic writer before broker integration.

## Usage
Construct with explicit topic and secured writer configuration at bootstrap.

## Validation
Run focused unit tests, then the Compose broker integration test.

## Elements
| Element | Behavior |
| --- | --- |
| `publisher.go` | Maps safe versioned outbox envelopes to `kafka-go` messages. |
| `publisher_test.go` | Specifies Kafka message mapping and failure handling. |
| `topic.go` | Idempotently applies required topic partitions and retention. |
| `topic_test.go` | Specifies idempotent topic configuration behavior. |
| `writer.go` | Builds bounded plaintext-local or TLS/SCRAM Kafka writers. |
| `writer_test.go` | Specifies broker validation and secure writer configuration. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
