---
name: tdd-development
description: "Enforce test-driven development for every repository change and planning activity. Use when planning, designing, implementing, modifying, debugging, or reviewing code: define testable acceptance criteria first, write focused failing tests before production behavior, use red-green-refactor, and do not treat manual or integration checks as substitutes for unit tests."
---

# TDD Development

## Planning

Before proposing or updating an implementation plan:

1. List the observable behaviors, errors, and edge cases.
2. Add a test-first checklist item for every planned production behavior.
3. Specify the test level: unit tests first; adapter, integration, or smoke tests only as additional wiring evidence.
4. Identify required fakes, fixtures, or contract suites. Keep valid and invalid test data as explicit, separate cases.

Do not plan production code without its corresponding test-first work.

## Implementation loop

For each behavior, complete this sequence in order:

1. Write the smallest focused test that expresses the expected behavior.
2. Run it and confirm that it fails for the expected missing or incorrect behavior.
3. Write the minimum production code needed to make it pass.
4. Run the relevant tests, then the full affected suite.
5. Refactor only while all tests remain green.

Do not write a handler, service, adapter, parser, configuration branch, error mapping, or bug fix before its focused test exists. Do not combine untested behavior with unrelated refactoring.

## Verification

- Test domain rules without infrastructure.
- Test application decisions with hand-written fakes.
- Test transport and storage mappings with focused adapter tests.
- Add integration or Docker smoke tests after unit tests; they verify wiring and never replace unit coverage.
- Test success, validation failure, dependency failure, duplicate/retry, and configuration failure paths when applicable.
- Treat a behavior without a practical automated test as blocked. Record the reason and obtain user direction before implementing it.

## Reporting

State the test written first, its initial failure, the implementation added, and the passing verification. Keep the repository plan's test-first checklist accurate as work progresses.
