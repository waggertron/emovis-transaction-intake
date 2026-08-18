# BUG-2026-08-18-003 — Lossy Producer Passthrough

**Status:** Fixed on `feat/openapi-contract-conformance`

## Observed behavior

`location` and `metadata` were decoded through `map[string]any`, converting JSON numbers to `float64`. Large identifiers could change value and producer formatting was lost before durable storage.

## Expected behavior

Arbitrary producer objects must retain exact audit bytes, and structured processing must not lose numeric precision.

## Root cause

The request boundary retained only parsed Go values and storage embedded them solely inside normalized JSON documents.

## Correction

The transport retains `json.RawMessage` bytes and parses structured values with `UseNumber`. NDJSON stores raw fields as base64 bytes, PostgreSQL uses separate `BYTEA` columns, DynamoDB uses binary attributes, and Kafka maps the raw objects without float conversion.

## Validation evidence

`TestIngestRetainsRawPassthroughBytes`, `TestIngestUsesNumberSafeArbitraryObjects`, adapter raw-persistence tests, and `TestPublisherEmitsNumberSafePassthroughFromRawAuditFields` cover the regression.

