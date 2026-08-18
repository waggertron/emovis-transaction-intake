# BUG-2026-08-18-005 — Ephemeral Default Compose Store

**Status:** Fixed on `feat/openapi-contract-conformance`

## Observed behavior

The default Compose application omitted `STORE_DRIVER`, so configuration selected memory storage. Restarting the application lost accepted transactions, idempotency state, and pending outbox work.

## Expected behavior

The primary local workflow must demonstrate durable acceptance and stable replay across application restart without cloud credentials.

## Root cause

Durable NDJSON was available only through a specialized E2E profile rather than the documented default stack.

## Correction

The default application selects NDJSON at `/data/transactions.ndjson`, uses a repository-scoped named volume, and depends on a least-privilege initialization container.

## Validation evidence

The container definition contract and `tests/smoke/local.sh` assert durable replay with the original transaction ID after restart and one Kafka event.

