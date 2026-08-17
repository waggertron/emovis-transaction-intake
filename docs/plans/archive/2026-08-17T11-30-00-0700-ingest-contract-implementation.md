# Supplied Transaction Ingest Contract Implementation

## Decisions

- User: adopt the supplied OpenAPI 3.0.3 contract.
- User: retain health, readiness, and metrics endpoints.
- User: make authentication configurable; local default is disabled.
- User: configure transaction types and default currency through environment variables.
- User: reject unknown top-level request fields while preserving arbitrary nested metadata and location.
- Codex: idempotency is `(source, source_reference)`; conflicting payloads return `400`.
- Codex: preserve exact decimal amount strings and generate the system transaction ID.

## Checklist

- [x] Create `feat/ingest-openapi-contract` from clean `main`.
- [x] Replace the mock OpenAPI contract with the supplied ingest contract.
- [x] Add contract tests for OpenAPI version, path, schemas, and statuses.
- [x] Update the domain and application model for source identity, decimal amounts, identifiers, statuses, metadata, and location.
- [x] Update HTTP routing, validation, configurable authentication, response mapping, and errors.
- [x] Update storage identity keys and PostgreSQL schema.
- [x] Update Kafka event mapping and local smoke/E2E request fixtures.
- [x] Update environment examples, Make targets, README, API instructions, and infrastructure notes.
- [x] Replace stale unit tests that asserted deleted routes and fields.
- [x] Raise HTTP, Kafka, PostgreSQL, application, and domain coverage to the repository's 85% gate.
- [x] Run full Compose validation and all E2E profiles.
- [ ] Commit, push, open the pull request, and archive this plan.

## Validation evidence

- `GOCACHE=/tmp/emovis-gocache go test ./...` passes.
- `GOCACHE=/tmp/emovis-gocache make lint` passes.
- `GOCACHE=/tmp/emovis-gocache bash tests/contracts/openapi_test.sh` passes.
- `GOCACHE=/tmp/emovis-gocache make validate` passes, including Docker, Compose, component, E2E, infrastructure, security, and image checks.
- Per-package coverage passes at least 85% for every production package.
