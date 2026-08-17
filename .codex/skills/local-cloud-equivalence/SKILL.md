---
name: local-cloud-equivalence
description: Design and validate production-adapter-backed local substitutes for cloud and third-party boundaries. Use when planning or implementing AWS services, databases, brokers, secret providers, object stores, queues, identity integrations, or tests that must remain confirmable without cloud credentials.
---

# Local Cloud Equivalence

1. Identify each external boundary and the production adapter that crosses it.
2. Select a deterministic local substitute that exercises that adapter's real protocol and failure mapping. Use fakes only for unit isolation; do not count them as component evidence.
3. Document the equivalence boundary: behaviors proved locally, cloud-specific properties proved statically, and behaviors requiring an optional real-cloud smoke test.
4. Provide one strict Make target per substitute plus an aggregate gate. Default development and CI to local substitutes without credentials.
5. Test success, validation, authentication, retries/duplicates, restart persistence where applicable, dependency unavailability, and cleanup.
6. Keep production and local configuration explicit. Reject unsafe fallback from a requested production provider to a fake.
7. Validate cloud configuration without apply: render manifests, validate provider schemas and policies, and use credential-free plans where supported.
8. Record unsupported parity honestly. Never claim that a local emulator proves managed-service IAM, quotas, availability, upgrades, or control-plane behavior.

Completion requires production-adapter behavior evidence, documented parity limits, and an audit showing no retained containers, volumes, networks, ports, credentials, or seeded state.
