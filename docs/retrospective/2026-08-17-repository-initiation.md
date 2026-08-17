# Repository initiation retrospective — 2026-08-17

## Outcome

The repository produced a production-shaped Go transaction-intake service with a mocked OpenAPI contract, interchangeable stores, transactional outbox, Kafka delivery, local cloud substitutes, AWS-oriented infrastructure, layered tests, security evidence, and a private GitHub handoff. The archived plan retains the complete decision and checklist history.

## What worked well

- **Decision history survived changing requirements.** Attributed notes and explicit supersession retained why Kafka replaced SNS/SQS, why the missing OpenAPI contract was mocked, and why repository visibility changed.
- **Ports and adapters paid for themselves.** One application contract supported memory, NDJSON, PostgreSQL, and DynamoDB without changing intake semantics.
- **Transactional outbox clarified reliability.** Atomic acceptance plus a stable event ID made database/Kafka failure and at-least-once delivery behavior explicit.
- **Local substitutes made completion independent of AWS.** PostgreSQL, DynamoDB Local, Kafka, secure Kafka, and a local secret provider exercised production adapters without cloud credentials.
- **Per-package coverage exposed hidden weakness.** The original aggregate percentage concealed weak command and adapter packages; an independent floor forced focused error-path tests.
- **Behavior matrices prevented percentage theater.** Critical assertions remained visible even after numerical coverage passed.
- **Fresh-clone and hosted checks found real onboarding defects.** They tested declared instructions rather than the accumulated state of the development machine.

## Rework and its causes

- The work began without the referenced OpenAPI attachment. Artifact inventory should have been the first hard gate; the later mock decision was correct but created avoidable uncertainty.
- The plan expanded repeatedly as coverage, component, E2E, security, infrastructure, and directory-instruction requirements were clarified. Atomic checklist items helped reconciliation, but a smaller initial scope statement would have reduced churn.
- Component and E2E aggregates duplicated expensive scenarios. Shared evidence was valuable, but future CI should avoid rerunning identical cloud-equivalence paths unless isolation requires it.
- The exact plain-language explanation duplicates parts of the architectural README. It is retained as requested and labeled point-in-time; stable design truth should continue to live in ADRs and focused documentation.

## Requirement changes worth remembering

- Missing source contract → explicitly mocked OpenAPI contract.
- SNS/SQS → Kafka locally and Amazon MSK in AWS.
- General test completeness → strict TDD, per-package 85% coverage, production-adapter component tests, and all-implementation E2E tests.
- Public repository handoff → private repository with explicit reviewer access.
- Completed active plan → archived plan, requiring a new current plan before later material changes.

## Native-Linux CI lessons

1. Developer utilities are not runner guarantees. The infrastructure contract used `rg`; hosted CI failed until the prerequisite was installed and contract-tested.
2. Docker Desktop is not proof of Linux bind-mount behavior. A mode-`0600` secret fixture was readable locally but not by the hosted container's different UID.
3. Secure fixes preserve controls. The solution aligned the non-root E2E UID/GID and used a capability-limited volume initializer rather than weakening secret permissions.
4. Generated metadata needs semantic checks. Shell expansion stripped `$skill-name` from scaffold prompts even though initialization otherwise succeeded.

## Abstractions that earned their cost

- `TransactionStore` as one cohesive atomic transaction/outbox port.
- Shared behavioral contracts across storage adapters.
- Separate API, worker, and combined-local composition modes.
- Make as the canonical command interface.
- Repository skills for TDD, decisions, Kafka, local/cloud equivalence, CI portability, coverage/behavior, and secure component/E2E testing.

## Intentional assignment overbuild

Terraform for full VPC/EKS/MSK/RDS/DynamoDB topology, secure Kafka with generated TLS/SCRAM material, every-directory `AGENTS.md`, and exhaustive cross-store E2E coverage exceed the minimum needed for a small take-home. They demonstrate platform breadth but would normally be prioritized with the team against delivery time and operational ownership.

## What to do differently with another four hours

1. Run artifact inventory and a native-Linux skeleton workflow before architecture implementation.
2. Establish a fixed scope budget and label demonstration-only infrastructure explicitly.
3. Design CI scenarios as a matrix with reusable artifacts to reduce duplicate Docker builds and repeated E2E paths.
4. Add generated-SDK or schema-conformance tooling only after receiving the actual OpenAPI contract.
5. Add benchmarks for intake throughput, outbox backlog recovery, and hot Kafka partition behavior instead of adding more infrastructure breadth.

## Durable follow-up

Use the [repository initiation checklist](../onboarding/repository-initiation-checklist.md) and `$bootstrap-production-service` skill for similar work. Start a new timestamped current plan before future material changes; do not reopen or rewrite the archived implementation plan.
