---
name: ci-portability
description: Make development and CI workflows reproducible across macOS, Docker Desktop, native Linux, and hosted runners. Use when adding or debugging workflows, shell scripts, containers, mounted fixtures, tool prerequisites, permissions, architecture differences, or commands that pass locally but fail in CI.
---

# CI Portability

1. Enumerate environmental assumptions before changing behavior: OS, architecture, shell, tool versions, filesystem ownership, Docker engine, network, credentials, and cache paths.
2. Declare every required tool in CI setup or a pinned container image. Add a workflow contract for nonstandard tools; never assume a developer utility such as `rg` exists on a runner.
3. Route local and hosted validation through the same Make targets.
4. Preserve non-root execution and restrictive file modes. For bind-mounted fixtures, align the test container UID/GID with the fixture owner and initialize writable volumes with only the capability required; do not make secrets world-readable to bypass ownership failures.
5. Avoid Docker Desktop-only evidence. Run container, bind-mount, line-ending, executable-bit, and case-sensitivity paths on native Linux CI early.
6. Keep generated paths inside declared writable locations and distinguish sandbox failures from repository defects.
7. Add a regression contract that fails before fixing a discovered portability issue.
8. Verify cleanup and a clean tracked worktree after repeated runs.

When a matrix is needed, vary one causal dimension at a time and record the passing/failing cells instead of applying speculative fixes.
