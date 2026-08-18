# Bug Record Instructions

## Purpose
Preserve reproducible defect reports, root-cause analysis, remediation evidence, and regression coverage.

## Scope
Applies to bug records under `docs/bugs/`.

## Local rules
- Use dated records with a stable bug identifier and explicit status.
- Separate observed behavior, expected behavior, root cause, correction, and validation evidence.
- Link to relevant source, tests, plans, commits, and hosted validation without including secrets or production data.

## Usage
Add a record when a confirmed defect materially violates the API contract, data integrity, security, or operational behavior.

## Validation
Run Markdown link validation and the AGENTS.md hierarchy validator.

## Elements
| Element | Behavior |
| --- | --- |
| `2026-08-18-idempotent-replay-transaction-id.md` | Records the fixed defect where an idempotent replay returned a newly generated transaction ID. |
| `2026-08-18-missing-required-fields-accepted.md` | Records missing required strings and implicit transaction-type defaulting. |
| `2026-08-18-lossy-producer-passthrough.md` | Records numeric precision and raw audit-byte loss across the intake/storage boundary. |
| `2026-08-18-openapi-schema-tightening.md` | Records incompatible tightening and alteration of the supplied contract. |
| `2026-08-18-ephemeral-default-compose-store.md` | Records loss of accepted transactions and outbox state across default Compose restart. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
