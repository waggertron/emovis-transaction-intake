# Replay Transaction ID Correction Plan — 2026-08-18

## Decision Notes

- **User:** Find and fix the API replay response returning a different transaction ID.
- **Codex:** Preserve the durable transaction ID in the storage outcome and use it in the application result for idempotent replays. This makes all storage adapters satisfy the supplied OpenAPI contract that a duplicate returns the existing record.

## Completion Checklist

1. [x] Reproduce the defect through the running HTTP API: an identical request returned `200 duplicate=true` with a different `id`.
2. [x] Identify the cause: replay outcomes omit the durable transaction ID, so the intake service returns a newly generated one.
3. [x] Write focused failing regression tests for the replay ID across application, in-memory, NDJSON, PostgreSQL, DynamoDB, and shared storage-contract layers. The pre-fix test failed because `StoreOutcome` had no transaction-ID field.
4. [x] Return the durable transaction ID from every storage implementation and use it in the application result. Smoke and E2E scripts now assert the replay ID equals the initial ID.
5. [x] Run focused application, adapter, HTTP contract, component, and E2E tests: passed. The Docker smoke path verifies `201` followed by `200` with the same transaction ID.
6. [x] Run the full local validation gate: `GOCACHE=/tmp/emovis-gocache make validate` passed, including unit, race, coverage, contracts, components, E2E, infrastructure, security, documentation, and hierarchy checks.
7. [x] Commit and push `43f40a2`, confirm the hosted CI run passed, and archive this plan.

## Validation Notes

- Codex's managed sandbox cannot write to the host Go build cache. Validation uses `GOCACHE=/tmp/emovis-gocache`; this is an execution-environment constraint, not a repository defect.
