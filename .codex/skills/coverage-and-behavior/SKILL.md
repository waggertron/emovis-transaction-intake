---
name: coverage-and-behavior
description: Define, implement, or review test coverage policies that combine per-package numerical floors with explicit critical-behavior evidence. Use when adding coverage gates, evaluating test completeness, building behavior matrices, preventing aggregate coverage from hiding weak packages, or assessing component paths that percentages represent poorly.
---

# Coverage and Behavior

1. Define production-package scope explicitly and fail when an expected package is absent or malformed.
2. Enforce the coverage floor independently per production package. Never let a high aggregate conceal a package below threshold.
3. Test the gate itself with missing data, stale profiles, excluded source, below-floor packages, and a false aggregate pass.
4. Create a reviewed behavior-to-test matrix covering correctness and failure risks: validation, authorization, idempotency, atomicity, concurrency, leases, retries, terminal outcomes, redaction, shutdown, and dependency failures where applicable.
5. Map every critical behavior to a named assertion. Add the missing test even when the percentage already passes.
6. Separate deterministic unit coverage from component evidence. Report real adapter coverage where meaningful and use behavioral reports for broker/process paths where one Go percentage is misleading.
7. Publish readable results under an ignored, self-cleaning artifact path.
8. Treat the threshold as a regression floor, not a quality score or incentive for low-value assertions.

Report numerical coverage and uncovered critical behavior separately.
