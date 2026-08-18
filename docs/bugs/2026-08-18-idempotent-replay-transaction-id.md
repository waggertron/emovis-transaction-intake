# BUG-2026-08-18-001: Idempotent Replay Returned a New Transaction ID

## Status

- **State:** Fixed
- **Detected:** 2026-08-18
- **Corrected:** 2026-08-18
- **Severity:** Medium — API contract and client-reference integrity violation; no duplicate billable record was created.
- **Affected endpoint:** `POST /ingest/v1/transactions`

## Summary

An identical retry correctly returned HTTP `200`, `duplicate=true`, and `Idempotent-Replay: true`, but its response contained a newly generated transaction `id` instead of the ID of the previously accepted record.

The durable store remained idempotent: it kept one transaction and one outbox event. The defect was confined to the replay response, where a producer could persist an ID that did not identify the durable transaction.

## Reproduction

1. Start the API with `make run-api`.
2. Submit a valid transaction to `/ingest/v1/transactions`.
3. Submit the identical payload again with the same `source` and `source_reference`.
4. Compare the `id` fields in the `201` and `200` response bodies.

Observed before the correction:

```text
first response:  201 duplicate=false id=ORBXSAXFIOBVT2ROYR5T6ML3RI
replay response: 200 duplicate=true  id=2ZAMVR7GKWS7MUYR6L4Z63QOUK
```

Expected behavior: both responses return the same durable transaction ID.

## Root Cause

The intake service generated a candidate transaction ID before every storage attempt. Storage adapters detected a replay and returned the original outbox event ID, but the storage outcome had no field for the original transaction ID. The application therefore built the replay result from the newly generated candidate ID.

Existing tests verified replay classification and preservation of the original event ID, but did not compare the first and replay transaction IDs. The smoke and E2E suites checked status, duplicate state, and the replay header without asserting ID stability.

## Correction

- [`StoreOutcome`](../../internal/transaction/app/intake.go) now carries the durable transaction ID.
- The intake service uses that durable ID for replay responses and rejects malformed replay outcomes that omit it.
- Memory and NDJSON stores retain the accepted transaction ID with the idempotency identity.
- PostgreSQL identity reads now select the stored transaction `id`.
- DynamoDB identity records now store and return `transaction_id`.
- Shared component contracts, adapter tests, smoke tests, and E2E tests require a replay to return the original transaction ID.

## Validation and Evidence

- Focused regression tests were written first and failed because `StoreOutcome` could not represent the durable transaction ID.
- `GOCACHE=/tmp/emovis-gocache make validate` passed locally, including unit, race, coverage, contract, component, E2E, infrastructure, security, documentation, and hierarchy gates.
- All production Go packages remained above the repository's 85% coverage requirement.
- Hosted CI passed for fix commit [`43f40a2`](https://github.com/waggertron/emovis-transaction-intake/commit/43f40a276a128f2012a63b7620919218d41af622): [GitHub Actions run](https://github.com/waggertron/emovis-transaction-intake/actions/runs/32152287740).
- The completed implementation checklist is archived in [the replay transaction-ID plan](../plans/archive/2026-08-18T15-15-00-0700-replay-transaction-id.md).

## Prevention

The storage port contract now includes transaction identity, and every local storage implementation must prove it preserves that identity across replay. Process-boundary smoke and E2E tests independently compare the response IDs, preventing a future adapter or HTTP-layer regression from passing on status codes alone.
