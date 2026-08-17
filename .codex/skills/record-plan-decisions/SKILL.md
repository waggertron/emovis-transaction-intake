---
name: record-plan-decisions
description: "Record material repository decisions in the active plan. Use whenever the user, Codex, or both choose, revise, supersede, or reject architecture, technology, security, testing, delivery, scope, or operational behavior that affects implementation or handoff."
---

# Record Plan Decisions

1. Locate the single active timestamped plan under `docs/plans/current/` before changing implementation.
2. Decide whether the statement is material: it changes scope, behavior, architecture, technology, security posture, validation, operations, or delivery expectations. Do not record routine mechanics or transient debugging observations.
3. Add a bullet to the plan's top-level `## Decision Notes` section before continuing the affected work.
4. Use this form: `**YYYY-MM-DD — Attribution — Decision.** Rationale. Implementation and validation impact.` Attribution is `User`, `Codex`, or `Joint`.
5. When a decision replaces an earlier one, retain the earlier history, label it superseded, and link the new decision explicitly. Never silently rewrite history.
6. Update affected numbered checklist items and acceptance criteria so they agree with the decision. Split mixed requirements instead of hiding incomplete work inside a checked item.
7. Run `git diff --check` and the plan/document validators. Report the recorded decision and any resulting checklist changes.

If no single active plan exists, stop implementation and report the repository-state inconsistency.
