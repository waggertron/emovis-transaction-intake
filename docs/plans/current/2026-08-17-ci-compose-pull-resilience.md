# CI Compose Pull Resilience Plan — 2026-08-17

## Decision Notes

- **User:** Fix the latest CI failure.
- **Codex:** The failed run was caused by Docker Hub returning HTTP 502 while Compose pulled DynamoDB Local, after all non-E2E jobs passed. Retry only the Compose startup operation, with a bounded attempt count and delay, so transient registry failures do not hide genuine test failures.

## Completion Checklist

1. [x] Identify the failed GitHub Actions job and preserve its evidence: the DynamoDB E2E startup pull returned HTTP 502.
2. [x] Write a failing test for bounded Compose-startup retry behavior.
3. [x] Add the retry helper and use it for E2E service startup.
4. [x] Add the helper test to the canonical E2E Make target and document it in directory instructions.
5. [x] Run the focused helper test and DynamoDB E2E test locally: both passed.
6. [x] Run the full local validation gate: `make validate` passed, including unit, race, 85% coverage, contracts, components, E2E, infrastructure, security, documentation, and hierarchy gates.
7. [ ] Commit, push, confirm hosted CI, then archive this plan.
