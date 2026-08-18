# Architecture Documentation

## Purpose

Maintain the durable, implementation-independent explanation of the transaction intake service's structure, boundaries, data flow, and deployment topology.

## Scope

Place architecture diagrams, component descriptions, dependency-direction rules, storage/outbox behavior, and local/cloud topology documents in this directory. Keep executable implementation, OpenAPI contracts, deployment manifests, and ADRs in their dedicated directories.

## Local rules

- Describe observable behavior and dependencies; do not duplicate source code line by line.
- Keep the domain/application core independent of HTTP, Kafka, storage, AWS, and configuration adapters.
- Document every external boundary's local confirmation path alongside its cloud-oriented design.
- Update this documentation when an architecture decision changes, and link the related ADR and current plan decision.

## Usage

Read this directory before adding a service boundary, storage adapter, Kafka workflow, local environment, or cloud deployment surface. Add a focused document when a diagram or explanation cannot fit in the current plan or ADR.

## Validation

- Verify diagrams, component names, dependency direction, and adapter lists match the current plan and source structure.
- Verify every local/cloud claim names a local confirmation path.
- Run the repository agent-instruction hierarchy validator after parent and child `AGENTS.md` files are present.

## Elements

| Element | Behavior |
| --- | --- |
| `2026-08-17-transaction-ingest-system.md` | Historical pre-conformance end-to-end system description. |
| `2026-08-18-openapi-contract-conformance.md` | Current architecture for exact contract handling, raw/canonical JSON, storage mappings, durable local defaults, and verification. |
| `transaction-and-kafka-flow.md` | Explains how a transaction moves from HTTP intake through durable storage and the outbox to Kafka, including the reasons for Kafka and its delivery guarantees. |

## Instruction hierarchy

- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none currently.
