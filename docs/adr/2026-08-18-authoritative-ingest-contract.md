# Authoritative Ingest Contract

## Problem Statement

The repository began from a mock contract and later received the actual OpenAPI 3.0.3 ingest specification. The mock tightened unknown-field handling, shortened descriptions, inlined named schemas, and mixed operational endpoints into the public producer contract. We need one stable authority while retaining useful operational behavior.

## Options Evaluated

### Option A: Preserve repository-specific tightening

| Pros | Cons |
| --- | --- |
| Retains strict parsing and one combined API document | Rejects schema-valid producer payloads and no longer implements the supplied contract |

### Option B: Preserve the supplied contract exactly and document extensions separately

| Pros | Cons |
| --- | --- |
| Produces deterministic conformance; avoids accidental breaking changes; keeps provenance clear | Requires separate documentation and tests for operational endpoints and implementation-only errors |

## Decision

Choose Option B. `api/openapi.yaml` is an exact copy of the supplied producer contract. Health, readiness, metrics, optional API-key authentication, body limits, media-type errors, conflicts, and dependency failures are repository extensions and must not silently alter that file.

## Implementation Details

- Preserve the supplied artifact at `docs/2026-08-18-supplied-openapi.yaml` for review provenance.
- Compare `api/openapi.yaml` byte-for-byte with that baseline in contract tests.
- Accept unspecified object properties because OpenAPI permits them by default.
- Test operational behavior through handler/component suites and document it in the README and dated architecture guide.

