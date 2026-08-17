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
| _No architecture documents yet_ | Future architecture documents explain one stable system boundary or topology. |

## Instruction hierarchy

- Parent: none currently; add a relative link to `docs/AGENTS.md` when that parent instruction file is created.
- Children: none currently.
