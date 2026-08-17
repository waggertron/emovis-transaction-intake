---
name: secure-component-e2e
description: Design, implement, or review secure component and end-to-end tests against real local service substitutes. Use for databases, Kafka, secret providers, API-to-worker flows, authentication, TLS/SASL, restart persistence, dependency failures, Docker Compose profiles, and teardown validation.
---

# Secure Component and E2E

## Component contract

1. Exercise the production adapter against the real local substitute.
2. Prove successful behavior, invalid configuration, authentication failure, unavailable dependency, retry/duplicate semantics, and persistence/concurrency behavior relevant to the boundary.
3. Keep ephemeral credentials obviously local, minimally scoped, redacted, and excluded from logs and Git.

## End-to-end contract

1. Enter through the production transport and cross the selected storage, worker/outbox, and broker boundaries.
2. Assert external outcomes, replay/conflict behavior, restart behavior, and dependency degradation rather than internal calls.
3. Test each supported local implementation independently; do not substitute a fake or silently skip a mode.

## Runtime safety

- Run application containers as non-root.
- Preserve TLS hostname and CA verification; test incorrect credentials and untrusted certificates.
- Keep secret fixtures restrictive and align non-root ownership instead of weakening permissions.
- Use unique Compose projects and temporary paths.
- Trap cleanup and assert removal of containers, volumes, networks, ports, certificates, credentials, files, and seeded state.
- Route focused and aggregate scenarios through explicit Make targets usable in hosted CI.
