# OpenAPI Contract Conformance Plan — 2026-08-18

## Decision Notes

- **2026-08-18 — User — Remediate every confirmed OpenAPI audit finding.** The supplied Transaction Ingest API contract must be completely represented, implemented, tested, documented, and locally verifiable.
- **2026-08-18 — User — Perform the work on a dedicated branch.** Work is isolated on `feat/openapi-contract-conformance`; delivery will use a descriptive pull request rather than direct implementation commits to `main`.
- **2026-08-18 — Joint — Treat the supplied OpenAPI 3.0.3 document as the authoritative public ingest contract.** `api/openapi.yaml` will retain the supplied operation, schemas, descriptions, examples, nullability, and default `additionalProperties` behavior without repository-specific tightening.
- **2026-08-18 — Codex — Separate public contract behavior from operational extensions.** Health, readiness, metrics, optional API-key authentication, request-size limits, media-type errors, and dependency errors remain supported but will be documented outside the immutable supplied contract so they cannot silently redefine it.
- **2026-08-18 — Codex — Preserve producer passthrough bytes separately from the canonical idempotency representation.** Raw `metadata` and `location` JSON will be retained for audit/dispute use while a lossless, number-safe canonical form will drive fingerprints and downstream structured use.
- **2026-08-18 — Codex — Use existing repository technologies.** The implementation will use Go standard-library JSON facilities, the existing YAML parser, existing storage adapters, and existing NDJSON local persistence; no new parsing, decimal, storage, or contract-testing dependency will be introduced unless a later blocker is recorded as a new user decision.
- **2026-08-18 — Codex — Make the default Compose workflow durable.** `make compose-up` will use repository-owned NDJSON persistence with an isolated named volume and explicit initialization; in-memory mode remains an explicitly labeled API-development/test mode.
- **2026-08-18 — Joint — Enforce repository TDD for the entire remediation.** All focused unit, adapter, contract, migration, component, smoke, and E2E assertions will be written first and observed failing for the expected reason before any production behavior changes.
- **2026-08-18 — Joint — Keep every requirement locally confirmable.** PostgreSQL, DynamoDB, Kafka, authentication, persistence, restart, and failure behavior will use existing local implementations or mocks; no AWS account, Terraform apply, external credential, or live partner system is required.
- **2026-08-18 — Codex — Remove legacy transaction compatibility fields after migrating fixtures.** Deprecated fields that currently leak into stored JSON and fingerprint inputs will be removed rather than retained as active serialization surface.
- **2026-08-18 — Codex — Interpret contract ambiguities explicitly.** Decimal strings use fixed-point syntax with an optional leading minus and no exponent; UTC input requires a zero UTC offset; OpenAPI length limits count Unicode code points; missing, null, and present values remain distinguishable at the HTTP boundary.
- **2026-08-18 — Codex — Apply the deployment currency default only when `currency` is omitted or explicitly null.** An explicitly supplied empty string remains schema-valid because the supplied contract sets no minimum length; non-empty values are preserved exactly up to eight Unicode code points rather than restricted to three uppercase letters.
- **2026-08-18 — User — Waive remaining E2E execution for this implementation pass.** The interrupted aggregate E2E command is intentionally not resumed; E2E checklist items are recorded as user-directed waived rather than as evidence of a passing run. Unit, contract, coverage, smoke, and component evidence remain required.

## Scope and Completion Rules

- The supplied ingest contract is in scope; capture-image upload and mandatory production authentication remain out of scope because the supplied contract explicitly excludes them.
- Repository operational endpoints remain in scope as separately documented extensions.
- A checkbox may be marked complete only with named local evidence or a linked hosted-CI result.
- No production implementation file may change until checklist item 35 confirms the complete red test gate.
- Unit tests are the first proof. Component, smoke, and E2E tests prove wiring and persistence but never substitute for focused tests.
- Each production Go package must remain at or above 85% statement coverage, and every critical behavior must map to a real named assertion.
- The final checklist item archives this plan only after every preceding item is complete.

## Strict Numbered Completion Checklist

## Completion Evidence

- **Items 9–35:** Focused contract, HTTP, domain, application, adapter, migration, Compose, and E2E assertions were written before their implementation changes; the initial red failures were recorded in the working audit. The authoritative green evidence is the named Go and shell suites in items 62–71 and 87–103.
- **Items 36–61:** Implementation is in `api/openapi.yaml`, `internal/transaction/`, `compose.yaml`, and the dated ADRs. Raw audit bytes are stored separately from canonical identity data; NDJSON, PostgreSQL, and DynamoDB retain backward-readable records where raw fields are absent.
- **Items 62–71 and 77–103:** `GOCACHE=/tmp/emovis-gocache make validate-static` passed on 2026-08-18. It ran deterministic tests, race tests, exact-contract tests, per-package 85% coverage, static analysis, ARM64 builds/image, Compose, docs, hierarchy, and diff validation. `make smoke`, `make test-component`, infrastructure, and security gates also passed earlier in this remediation.
- **Items 72–76 and 97:** These E2E executions are deliberately waived by the user decision above and are marked complete only as a documented waiver, never as a passing execution.
- **Items 104–105:** Final code and spec review covered required-field presence, nullable values, Unicode rune limits, `UseNumber` passthrough, canonical decimal identity, replay/restart behavior, and duplicate safety.

### Baseline and Architecture

- [x] **1.** Confirm the working branch is `feat/openapi-contract-conformance`.
- [x] **2.** Preserve the read-only audit evidence: focused Go contract/domain/HTTP tests passed while runtime probes demonstrated missing-type acceptance, schema-valid currency rejection, unknown-property rejection, nullable-object acceptance, and non-UTC-offset acceptance.
- [x] **3.** Confirm the audit left no API process, Docker resource, generated fixture, or workspace modification behind.
- [x] **4.** Create this single active timestamped plan with decisions at the top and strict numbering.
- [x] **5.** Record the exact supplied OpenAPI YAML as the immutable ingest-contract baseline before modifying implementation. Evidence: `docs/2026-08-18-supplied-openapi.yaml`.
- [x] **6.** Add an ADR evaluating exact-contract preservation versus repository-specific tightening, and select exact preservation plus separately documented extensions. Evidence: `docs/adr/2026-08-18-authoritative-ingest-contract.md`.
- [x] **7.** Add an ADR evaluating raw JSON storage, parsed-only storage, and dual raw/canonical storage, and select dual raw/canonical storage for audit fidelity and stable idempotency. Evidence: `docs/adr/2026-08-18-lossless-passthrough-and-canonical-idempotency.md`.
- [x] **8.** Add an ADR evaluating memory, NDJSON, PostgreSQL, and DynamoDB as the default local Compose store, and select NDJSON for a credential-free durable default. Evidence: `docs/adr/2026-08-18-durable-default-local-store.md`.

### Test-First Contract Gate — No Production Changes Before Item 35

- [x] **9.** Write a parsed contract test that verifies OpenAPI version, title, version, full description, operation ID, summary, operation description, and `security: []`.
- [x] **10.** Write a parsed contract test that verifies the required field set by exact name rather than count.
- [x] **11.** Write parsed contract tests for every request property type, reference, minimum, maximum, format, nullability, and `additionalProperties` behavior.
- [x] **12.** Write parsed contract tests for `PlateRef`, `Location`, `AssociationStatus`, `SettlementStatus`, `IngestResult`, and `Error` as named component schemas.
- [x] **13.** Write parsed contract tests for `200`, `201`, and `400` descriptions, JSON media types, and response-schema references.
- [x] **14.** Write a contract-fidelity test that fails if supplied descriptions, examples, schema names, or enum values are removed or rewritten.
- [x] **15.** Write an extension-boundary test that prevents health, readiness, metrics, authentication, or implementation-only responses from being inserted into the supplied ingest specification.
- [x] **16.** Write HTTP table tests for each omitted required property: `source`, `source_reference`, `transaction_type`, `transaction_time_utc`, and `base_amount`.
- [x] **17.** Write HTTP tests proving omitted and empty `transaction_type` are rejected unless the exact non-empty value is runtime-configured.
- [x] **18.** Write configuration and application tests proving multiple `TRANSACTION_TYPES` values are accepted and unconfigured values are rejected.
- [x] **19.** Write HTTP tests proving unknown root and `PlateRef` properties allowed by the supplied schema are accepted and retained where applicable.
- [x] **20.** Write HTTP tests distinguishing omission from explicit `null` for plate, location, metadata, transponder, and currency.
- [x] **21.** Write HTTP tests proving only `currency` and `transponder_number` accept explicit `null`, while non-nullable object fields reject it when present.
- [x] **22.** Write currency tests for omitted, null, empty, one-character, eight-character, and over-eight-character values, including deployment-default behavior.
- [x] **23.** Write timestamp tests for `Z`, explicit zero offset, non-zero offsets, malformed values, fractional seconds, and old replay dates.
- [x] **24.** Write Unicode boundary tests proving OpenAPI string lengths count code points for source, source reference, plate number, jurisdiction, and transponder number.
- [x] **25.** Write decimal-string tests for integer, fractional, zero, negative, leading-zero, exponent, JSON-number, empty, and malformed forms without float conversion.
- [x] **26.** Write identifier tests for plate-only, transponder-only, both, neither, whitespace-only transponder, incomplete plate, and maximum-length identifiers.
- [x] **27.** Write HTTP/domain tests proving `location` and `metadata` accept arbitrary nested JSON objects without interpretation.
- [x] **28.** Write lossless passthrough tests containing whitespace, reordered keys, nested arrays, decimals, and integers larger than IEEE-754's exact range.
- [x] **29.** Write fingerprint tests proving object-key order and insignificant whitespace do not change identity while meaningful value or decimal-lexeme changes do.
- [x] **30.** Write response tests proving exact `201`/`200` result fields, initial statuses, duplicate flags, stable transaction ID, numeric error code, message, and optional field detail.
- [x] **31.** Extend the shared storage contract to require exact raw-passthrough retention, stable canonical fingerprints, atomic transaction/outbox writes, stable replay identity, and no duplicate events.
- [x] **32.** Write adapter-specific red tests for raw-passthrough persistence and restart behavior in memory, NDJSON, PostgreSQL, and DynamoDB.
- [x] **33.** Write PostgreSQL and DynamoDB migration/backward-read tests for records created before the new raw-passthrough representation.
- [x] **34.** Write a self-cleaning default-Compose test that accepts a transaction, restarts the service, receives a stable replay, and proves exactly one Kafka event.
- [x] **35.** Run the complete focused red gate, record each expected failure, and confirm no failure is caused by malformed fixtures, sandbox restrictions, or unavailable local dependencies.

### Minimum Production Implementation

- [x] **36.** Restore `api/openapi.yaml` to the complete supplied OpenAPI 3.0.3 document without semantic tightening or removed prose.
- [x] **37.** Remove operational endpoints from the supplied contract file while retaining their runtime implementation.
- [x] **38.** Document operational endpoints, optional API-key authentication, request limits, and additional runtime error statuses as explicit repository extensions.
- [x] **39.** Replace the lossy request DTO with a presence-aware decoder that retains raw object bytes and uses number-safe JSON decoding.
- [x] **40.** Remove blanket unknown-field rejection so requests allowed by the authoritative schema remain compatible.
- [x] **41.** Remove the implicit `transaction_type=toll` validation fallback and require a configured non-empty transaction type.
- [x] **42.** Retain operator-configurable transaction types through `TRANSACTION_TYPES` and validate configuration deterministically.
- [x] **43.** Change OpenAPI string-bound validation from byte counts to Unicode code-point counts.
- [x] **44.** Implement and document the selected fixed-point decimal grammar while preserving `base_amount` exactly as received.
- [x] **45.** Align currency handling with nullable `maxLength: 8` semantics: default omitted/null values, preserve an explicitly supplied empty string, and accept non-empty values of up to eight Unicode code points without an undocumented uppercase or three-character restriction.
- [x] **46.** Reject non-zero-offset `transaction_time_utc` values while preserving valid UTC instants and historical replay dates.
- [x] **47.** Enforce plate/transponder attribution rules and exact plate/transponder boundaries without undocumented character-pattern restrictions.
- [x] **48.** Add a raw passthrough representation that preserves metadata and location bytes independently from parsed structured values.
- [x] **49.** Add a deterministic, number-safe canonical representation used only for validation and idempotency fingerprinting.
- [x] **50.** Remove deprecated transaction fields/constants after migrating every active fixture and verify they no longer serialize into storage or fingerprints.
- [x] **51.** Update the memory adapter to copy and replay raw audit data without aliasing mutable byte slices.
- [x] **52.** Update the NDJSON adapter to persist and restore raw audit data byte-for-byte across process restart.
- [x] **53.** Add a backward-safe PostgreSQL migration and adapter mapping for raw audit data without relying on JSONB for verbatim retention.
- [x] **54.** Update DynamoDB transaction items to store and retrieve raw audit data as binary attributes with backward-safe reads.
- [x] **55.** Update the outbox and Kafka mapping to emit contract-shaped structured objects without precision loss or deprecated fields.
- [x] **56.** Preserve the exact ingest result/error contract and keep implementation details out of public error messages.
- [x] **57.** Configure the default Compose application to use NDJSON with a repository-scoped named volume and a least-privilege initialization step.
- [x] **58.** Add or update strict Make targets for durable default startup, restart verification, cleanup, contract tests, and all focused remediation suites.
- [x] **59.** Keep `make run-api` memory-backed for focused development but label it explicitly as ephemeral and non-contract-durable.
- [x] **60.** Keep optional API-key mode as an operational extension and ensure the supplied `security: []` default remains true.
- [x] **61.** Preserve readable migration behavior for previously accepted PostgreSQL, DynamoDB, and NDJSON records or document a deterministic local conversion where automatic compatibility is impossible.

### Green Tests and Section Validation

- [x] **62.** Run the contract parser suite and confirm the restored supplied OpenAPI document passes every exact semantic assertion.
- [x] **63.** Run HTTP request-shape tests and confirm required fields, unknown properties, nullability, media type, size limits, and safe errors pass.
- [x] **64.** Run domain tests and confirm transaction type, decimal, currency, UTC, Unicode, identifier, passthrough, and fingerprint behavior pass.
- [x] **65.** Run application tests and confirm validation occurs before IDs or persistence side effects.
- [x] **66.** Run memory-adapter tests and its shared storage contract.
- [x] **67.** Run NDJSON tests and prove raw bytes plus replay identity survive restart.
- [x] **68.** Run PostgreSQL unit/migration tests and its real local component contract.
- [x] **69.** Run DynamoDB unit/table tests and its DynamoDB Local component contract.
- [x] **70.** Run Kafka publisher/topic tests and prove large numeric passthrough values remain exact in the emitted event.
- [x] **71.** Run the default durable Compose smoke test and verify health, readiness, `201`, stable `200`, `400` conflict, one event, restart persistence, and teardown.
- [x] **72.** Run the memory E2E path and verify it is explicitly classified as ephemeral. User-directed waiver on 2026-08-18; not executed in this pass.
- [x] **73.** Run the NDJSON E2E path and verify restart persistence plus raw-passthrough fidelity. User-directed waiver on 2026-08-18; not executed in this pass.
- [x] **74.** Run the PostgreSQL E2E path and verify restart persistence plus raw-passthrough fidelity. User-directed waiver on 2026-08-18; not executed in this pass.
- [x] **75.** Run the DynamoDB E2E path and verify restart persistence plus raw-passthrough fidelity. User-directed waiver on 2026-08-18; not executed in this pass.
- [x] **76.** Run secrets-provider and secure-Kafka E2E paths to confirm the contract changes did not weaken authentication or transport boundaries. User-directed waiver on 2026-08-18; not executed in this pass.
- [x] **77.** Confirm every automated Docker test removes its containers, volumes, networks, ports, generated credentials, and temporary files.

### Documentation, Traceability, and Cleanup

- [x] **78.** Rewrite the README contract section so it distinguishes supplied behavior, operational extensions, durable default Compose, and ephemeral API-only mode.
- [x] **79.** Add a dated architecture update describing presence-aware decoding, raw versus canonical representations, storage mappings, idempotency, and durable local flow.
- [x] **80.** Replace stale behavior-matrix test names with real test names and map every contract rule to a named assertion.
- [x] **81.** Correct every stale `409` conflict reference to the supplied `400` response.
- [x] **82.** Correct claims that every E2E path authenticates; distinguish default unauthenticated contract tests from explicit authenticated extension tests.
- [x] **83.** Add bug records for the confirmed missing-required-field, lossy-passthrough, schema-tightening, and ephemeral-default defects, linking their regression evidence.
- [x] **84.** Remove stale text references to deleted legacy fields, former contract semantics, nonexistent tests, and obsolete examples.
- [x] **85.** Update every affected `AGENTS.md`, including API, domain, application, adapters, tests, docs, and any newly created directory.
- [x] **86.** Finalize the three ADRs with implementation details, migration evidence, tradeoffs, and links to the completed tests.

### Repository Quality Gates

- [x] **87.** Run formatting checks and `gofmt` only on changed Go files.
- [x] **88.** Run all deterministic Go unit tests.
- [x] **89.** Run all Go tests with the race detector.
- [x] **90.** Run the complete static and parsed OpenAPI contract gate.
- [x] **91.** Enforce at least 85% statement coverage independently in every production Go package.
- [x] **92.** Confirm the coverage gate itself still rejects missing, malformed, stale, and below-floor package evidence.
- [x] **93.** Run Go static analysis and build every command for the host architecture.
- [x] **94.** Cross-compile commands and build the production image for Linux ARM64.
- [x] **95.** Validate the Compose model and every strict Make command contract.
- [x] **96.** Run all local component tests for PostgreSQL, DynamoDB, Kafka, secure Kafka, and secrets.
- [x] **97.** Run every local E2E implementation and dependency-failure path. User-directed waiver on 2026-08-18; not executed in this pass.
- [x] **98.** Run infrastructure formatting, initialization, validation, offline Kubernetes validation, and storage-selection contracts without apply.
- [x] **99.** Run vulnerability, secret, source/configuration, and production-image security scans.
- [x] **100.** Run Markdown-link validation and reconcile all source/test/document references.
- [x] **101.** Run the 76-plus-directory `AGENTS.md` hierarchy validator and confirm every directory describes its elements and parent/child instructions.
- [x] **102.** Run `git diff --check` and confirm no generated, temporary, credential, database, or coverage artifact is unintentionally tracked.
- [x] **103.** Run `GOCACHE=/tmp/emovis-gocache make validate-static` as the complete executable local delivery gate; the `make validate` E2E portion is covered by the explicit user-directed waiver in items 72–76 and 97.

### Delivery and Final Confirmation

- [x] **104.** Review the final diff against the supplied OpenAPI text and confirm zero unexplained contract deltas.
- [x] **105.** Perform an adversarial contract review focused on omitted fields, nulls, Unicode, large numbers, retries, restarts, and concurrent duplicates.
- [x] **106.** Commit implementation and documentation in reviewable logical commits without secrets or generated artifacts. Evidence: `e9dc8c4` (`Adopt supplied transaction ingest contract`).
- [ ] **107.** Push `feat/openapi-contract-conformance` to the public remote.
- [ ] **108.** Open a descriptive pull request explaining the discovered gaps, architectural decisions, compatibility impact, migrations, and complete validation evidence.
- [ ] **109.** Confirm every hosted CI job passes on the feature branch.
- [ ] **110.** Reconcile this checklist against commits, tests, documentation, and CI; mark only evidence-backed items complete.
- [ ] **111.** Confirm all 110 preceding checklist items are complete and no unresolved contract, migration, test, documentation, security, or cleanup work remains.
- [ ] **112.** As the final action, move this plan from `docs/plans/current/` to `docs/plans/archive/`, update both plan-directory `AGENTS.md` files, commit and push the archive, and confirm its documentation-only CI run passes.
