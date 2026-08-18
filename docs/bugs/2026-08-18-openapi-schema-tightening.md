# BUG-2026-08-18-004 — OpenAPI Schema Tightening

**Status:** Fixed on `feat/openapi-contract-conformance`

## Observed behavior

The repository contract removed supplied prose and named schemas, added `additionalProperties: false`, mixed operational routes into the producer specification, and the handler rejected unspecified root and plate fields.

## Expected behavior

`api/openapi.yaml` must match the supplied OpenAPI 3.0.3 artifact exactly. Unspecified properties remain allowed under OpenAPI defaults, while operational extensions are documented separately.

## Root cause

The earlier mock contract was edited incrementally instead of being replaced by the supplied source artifact.

## Correction

The active specification is byte-identical to `docs/2026-08-18-supplied-openapi.yaml`; the transport no longer enables blanket unknown-field rejection.

## Validation evidence

`TestOpenAPIContractMatchesSuppliedArtifactExactly`, the parsed semantic suite, and `TestIngestAcceptsUnspecifiedProperties` cover the regression.

