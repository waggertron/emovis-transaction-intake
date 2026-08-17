# Secure Kafka Fixture Instructions

## Purpose
Generate ephemeral TLS and SCRAM state for the isolated secure Kafka profiles.

## Scope
Applies to secure Kafka bootstrap fixtures in this directory.

## Local rules
- Store generated keystores, truststores, private keys, and client files only in named test volumes.
- Preserve hostname verification and use SCRAM-SHA-512.
- Test credentials are local-only and must be removed during teardown.

## Usage
The `kafka-secure-init` Compose service runs the bootstrap script before the broker.

## Validation
Run secure Kafka component and E2E targets, including wrong-credential and untrusted-certificate failures.

## Elements
| Element | Behavior |
| --- | --- |
| `init.sh` | Generates ephemeral stores, client configuration, and SCRAM metadata. |
| `server.properties` | Supplies the minimal matching KRaft format configuration. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
