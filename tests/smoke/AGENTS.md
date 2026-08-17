# Smoke Test Instructions

## Purpose
Exercise the complete local API, outbox, and Kafka flow.

## Scope
Applies to self-cleaning end-to-end smoke scripts.

## Local rules
- Start and stop only the named Compose project.
- Use explicit fixtures and verify acceptance, replay, conflict, and event publication.
- Leave no containers, volumes, networks, ports, or temporary files.

## Usage
Run through `make smoke` with Docker available.

## Validation
Confirm the script exits zero and no project containers remain.

## Elements
| Element | Behavior |
| --- | --- |
| `local.sh` | Runs and cleans the complete local transaction flow. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
