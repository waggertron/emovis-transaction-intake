---
name: bootstrap-production-service
description: Bootstrap or assess a production-shaped service repository with decision-led planning, test-first delivery, local external-service substitutes, security evidence, agent instructions, and a reproducible handoff. Use when starting a backend take-home, microservice, API repository, platform service, or major greenfield subsystem.
---

# Bootstrap Production Service

1. Inventory supplied artifacts before selecting architecture. Label missing contracts and repository-authored assumptions.
2. Confirm repository ownership, remote, visibility, delivery method, and destructive/external-action boundaries.
3. Initialize version control and create one timestamped active plan with attributed, supersedable decisions and a complete numbered checklist.
4. Define observable acceptance behavior, test layers, per-package coverage policy, security review, and local confirmation requirements before production code.
5. Select technologies from explicit constraints; ask only when a choice materially changes the result.
6. Design domain/application ports before adapters. Define atomicity, idempotency, retry, delivery, and failure semantics explicitly.
7. Apply `$tdd-development` for every behavior and retain red-green evidence.
8. Apply `$local-cloud-equivalence`, `$coverage-and-behavior`, `$secure-component-e2e`, and `$ci-portability` while building local, hosted, and cloud-shaped validation.
9. Provide one strict Make interface, reproducible containers, least-privilege CI, security scans, architecture/operations documentation, and linked `AGENTS.md` instructions.
10. Run static, unit, race, contract, component, E2E, smoke, security, documentation, hierarchy, and cleanup gates as applicable.
11. Test a fresh clone using only declared onboarding instructions.
12. Reconcile every checklist item with evidence, record residual limits, confirm repository access, archive the completed plan last, and hand off the URL.

Do not broaden a small service merely to demonstrate technology. Document intentional omissions and prefer coherent production behavior over feature count.
