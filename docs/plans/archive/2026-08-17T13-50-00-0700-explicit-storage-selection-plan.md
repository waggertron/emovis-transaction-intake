# Explicit Terraform storage-selection plan

## Decision Notes

- **2026-08-17 — User — Do not enable both Terraform persistence implementations implicitly.** A deployment must require an explicit user decision for DynamoDB or PostgreSQL.
- **2026-08-17 — Codex — Use selectable Terraform modules instead of commenting resources out.** Shared resources remain common; exactly one persistence module is instantiated from a required `storage_backend` value.
- **2026-08-17 — Joint — Make the plan command fail without a selection.** `make terraform-plan` requires `TFVARS=dynamodb.tfvars.example` or `TFVARS=postgres.tfvars.example`; no apply target is introduced.

## Completion checklist

- [x] **1.1** Add the required no-default `storage_backend` variable and validation.
- [x] **1.2** Move shared secrets, DynamoDB, and PostgreSQL resources into documented modules.
- [x] **1.3** Add explicit backend variable examples and require `TFVARS` in the Make plan command.
- [x] **1.4** Make outputs and workload IAM conditional on the selected backend.
- [x] **1.5** Add tests proving missing selection fails and each plan excludes the unselected database.
- [x] **1.6** Document the decision in the README, infrastructure reference, ADR, and module instructions.
- [x] **1.7** Run Terraform, infrastructure, contract, full validation, documentation, hierarchy, and diff checks; archive this plan last.

## Execution evidence

- Terraform formatting and validation pass with no provider deprecation warnings.
- `make terraform-plan` without `TFVARS` fails with an actionable selection error.
- DynamoDB example plan contains `module.dynamodb[0]` and no PostgreSQL instance; PostgreSQL example plan contains `module.postgres[0]` and no DynamoDB table.
- `make test-infrastructure`, `make test-contract`, and `make validate` exit successfully.
- Documentation links, 75-directory AGENTS hierarchy, and `git diff --check` pass; no repository-owned Docker containers, volumes, or networks remain.
