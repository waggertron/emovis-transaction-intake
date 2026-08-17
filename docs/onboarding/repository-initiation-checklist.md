# Repository initiation checklist

Use this checklist for a new production service or substantial greenfield subsystem. Tailor it to explicit requirements; do not add technology solely to demonstrate it.

## 1. Inventory the starting material

- Confirm every referenced contract, attachment, diagram, credential expectation, and delivery instruction actually exists.
- Record file sizes, formats, encodings, timestamps, and provenance when source artifacts matter.
- Clearly label missing inputs and repository-authored mocks or assumptions.

## 2. Establish repository boundaries

- Initialize version control and the default branch before implementation.
- Confirm repository owner, remote, visibility, reviewer access, delivery method, and whether commits/pushes are authorized.
- Preserve unrelated user changes and define generated/local-only paths immediately.

## 3. Create the decision-led plan

- Create one timestamped plan under `docs/plans/current/`.
- Put attributed decisions at the top with date, rationale, impact, and explicit supersession.
- Define the complete, strictly numbered, atomic acceptance checklist before production code.

## 4. Define behavior and test policy

- List observable success, validation, authorization, duplicate, dependency-failure, concurrency, retry, and shutdown behavior.
- Require focused red-green-refactor evidence for every production behavior.
- Define per-package coverage floors plus a critical behavior-to-test matrix.
- Separate unit, contract, component, E2E, smoke, security, and fresh-clone evidence.

## 5. Select architecture and technology

- Apply explicit stack constraints first; ask when an unresolved choice materially changes delivery.
- Define domain/application boundaries, ports, atomicity, idempotency, and delivery semantics before adapters.
- Record major alternatives and consequences in ADRs.

## 6. Make external boundaries local-first

- Provide a production-adapter-backed local substitute for every database, broker, secret provider, queue, or cloud service needed for completion.
- Document what the substitute proves and which managed-service properties remain unproved.
- Keep real-cloud smoke tests optional, explicitly targeted, and non-destructive unless separately authorized.

## 7. Build through one strict interface

- Expose supported build, run, test, validation, security, infrastructure, and cleanup workflows through Make targets.
- Containerize runnable components with non-root, reproducible images.
- Declare every CI prerequisite and execute the same Make targets locally and on native Linux CI.

## 8. Secure and operate the service

- Define authentication, authorization, secret handling, redaction, body limits, data classification, encryption, network boundaries, and least privilege.
- Define health, readiness, metrics, logs, capacity signals, retry exhaustion, and incident-relevant failure modes.
- Run dependency, secret-history, source/IaC, image, and concurrency scans appropriate to the stack.

## 9. Validate the repository itself

- Maintain linked `AGENTS.md` instructions for every authored directory and accurate direct-element behavior.
- Validate skills, links, OpenAPI/contracts, infrastructure without apply, formatting, tests, coverage, cleanup, and a clean worktree.
- Run container and mounted-file flows on native Linux early to expose ownership and tool-availability assumptions.

## 10. Prove handoff quality

- Test a fresh clone using only declared onboarding instructions.
- Confirm CI is green, reviewer access matches repository visibility, no secrets/generated artifacts are tracked, and every plan item has evidence.
- Record residual limits, archive the completed plan last, and hand off the repository URL.
