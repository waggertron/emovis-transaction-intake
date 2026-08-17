# Explicit storage infrastructure selection

## Problem Statement

The reference Terraform configuration previously planned both DynamoDB and PostgreSQL even though the application uses one transaction store at a time. That creates unnecessary cost and operational surface and turns an unresolved product/platform choice into an implicit deployment decision. Terraform must remain reviewable and locally valid without silently choosing or creating both stores.

## Options Evaluated

### Option A: Comment out unselected resources

| Pros | Cons |
| --- | --- |
| Visually removes resources from a plan. | Produces configuration drift, weakens validation, obscures intent, and requires editing source to change choices. |

### Option B: Keep both stores enabled

| Pros | Cons |
| --- | --- |
| Simplest Terraform graph and validates both implementations together. | Plans unnecessary infrastructure and makes a costly architecture choice implicitly. |

### Option C: Require an explicit backend and instantiate selectable modules

| Pros | Cons |
| --- | --- |
| Preserves both reviewed implementations, validates module contracts, prevents a default choice, creates only the selected store, and permits least-privilege IAM. | Adds module interfaces and requires the operator to choose a variable file before planning. |

## Decision

Choose Option C. `storage_backend` has no default and accepts only `dynamodb` or `postgres`. The root configuration always owns shared platform resources and secret containers, but uses conditional module instances so exactly one persistence implementation appears in a plan. A missing selection is an intentional error, not a fallback.

This keeps the decision with the user/operator while retaining executable, reviewable Terraform. It also ensures DynamoDB permissions are omitted from the workload policy when PostgreSQL is selected.

## Implementation Details

- `infra/terraform/modules/shared` owns storage-independent API and Kafka secret containers.
- `infra/terraform/modules/dynamodb` owns the encrypted transaction/outbox table and dispatch index.
- `infra/terraform/modules/postgres` owns the private subnet group, PostgreSQL secret container, and hardened RDS instance.
- `dynamodb.tfvars.example` and `postgres.tfvars.example` are the only checked-in selection examples.
- `make terraform-plan` requires `TFVARS=dynamodb.tfvars.example` or `TFVARS=postgres.tfvars.example`; no Make target applies a plan.
- Root outputs return `null` for the unselected backend, and IAM includes only resources needed by the selected implementation.
- Infrastructure contract tests reject a default selection, root-owned database resources, missing examples, and deprecated DynamoDB index syntax.
