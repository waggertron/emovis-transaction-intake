# AWS infrastructure reference

The reference topology is intentionally non-applying. CI and `make validate` may format, initialize, validate, render, and create a credential-free Terraform plan; they never run `terraform apply` or contact a Kubernetes cluster.

## Prerequisites

Local validation requires Terraform 1.8+, `kubectl` with Kustomize support, Python 3 with PyYAML, Docker Compose, Go, and Make. A real deployment additionally requires an explicitly selected non-production AWS account, suitable quotas, an approved remote-state design, Route 53/TLS ingress decisions, immutable ECR image digests, and an operator-authorized apply workflow.

## Terraform inputs

| Variable | Default | Purpose |
| --- | --- | --- |
| `aws_region` | `us-west-2` | AWS region for the topology. |
| `availability_zones` | three west-2 AZs | Spreads private/public subnets and data services. |
| `name` | `emovis-transaction-intake` | Resource naming prefix. |
| `vpc_cidr` | `10.42.0.0/16` | VPC address space. |
| `environment` | `nonprod` | Safety/environment tag. |
| `eks_version` | `1.33` | EKS control-plane version; verify support before apply. |
| `kafka_version` | `3.8.x` | MSK broker version; verify regional support before apply. |
| `db_instance_class` | `db.t4g.small` | Non-production PostgreSQL size. |
| `msk_instance_type` | `kafka.t3.small` | Non-production MSK broker size. |
| `deletion_protection` | `true` | Protects RDS and requires a final snapshot; only the non-applying example opts out. |
| `local_validation` | `true` | Enables credential-free provider behavior for local planning only. |
| `storage_backend` | none; required | Explicitly selects exactly one of `dynamodb` or `postgres`; Terraform never chooses a store implicitly. |
| `runtime_secrets_ready` | `false` | Confirms externally populated runtime/SCRAM secrets before association and topic bootstrap are enabled. |
| `tags` | project defaults | Additional resource ownership metadata. |
| `topic_name` | review-candidates v1 | Kafka topic configured by the post-MSK Job. |
| `topic_partitions` / `topic_replication` | `3` / `3` | Production topic distribution and durability. |
| `topic_retention` | `168h` | Broker retention passed to the bootstrap command. |
| `topic_bootstrap_image` | immutable placeholder | Digest-pinned Job image that must be replaced before apply. |

Choose either `infra/terraform/dynamodb.tfvars.example` or `infra/terraform/postgres.tfvars.example` only for an authorized workflow; review cost and deletion settings before changing `local_validation`. The checked-in provider lock file makes local and CI validation reproducible.

## Topology and identity

The VPC spans three availability zones. EKS, MSK, and the explicitly selected DynamoDB or multi-AZ RDS backend are private; the EKS public endpoint and unauthenticated MSK access are disabled. KMS protects EKS secrets, MSK storage, the selected database, and Secrets Manager. Broker logs and a CPU alarm provide a minimum operational surface.

The API and worker use the `transaction-intake` Kubernetes service account. Its IRSA trust is restricted to that exact namespace/name and grants DynamoDB transaction-table operations, reads of the named runtime secrets, and decrypt access to the data KMS key. Node credentials are not the workload identity.

Application pods and topic bootstrap set `AWS_SECRET_ID=emovis-transaction-intake/api`; each Go composition root retrieves its bounded JSON map through Secrets Manager using IRSA. Explicit environment values override provider values. A production secret can supply `API_KEY`, `PARTNER_ID`, `POSTGRES_URL`, `KAFKA_BROKERS`, `KAFKA_SASL_USERNAME`, and `KAFKA_SASL_PASSWORD`. MSK's associated SCRAM secret remains the broker-side credential record and must be kept consistent through the authorized secret-provisioning workflow.

Secret provisioning is deliberately two-stage: first apply with `runtime_secrets_ready=false` to create the secret containers and infrastructure, populate the API and `AmazonMSK_` SCRAM secrets through the authorized workflow, then set `runtime_secrets_ready=true` for the reviewed follow-up plan/apply. Terraform will not attempt SCRAM association or start the topic Job before that explicit confirmation.

Terraform has no default persistence backend. Its root module instantiates exactly one child module based on the required `storage_backend` input. `dynamodb` creates the transaction/outbox table and grants its scoped workload permissions; `postgres` creates private RDS and its secret container while omitting DynamoDB permissions. `STORE_DRIVER` and the runtime secret must match that explicit infrastructure choice. The checked-in Kubernetes reference remains an illustrative DynamoDB manifest and uses immutable placeholder digests that must be replaced before deployment.

## Local validation

Use only the canonical commands:

```bash
make terraform-fmt
make terraform-validate
make terraform-plan TFVARS=dynamodb.tfvars.example
# or
make terraform-plan TFVARS=postgres.tfvars.example
make k8s-validate
make test-infrastructure
```

Calling `make terraform-plan` without `TFVARS` fails intentionally before planning. Each selected example uses `-refresh=false`, no backend, and mock credentials. Kubernetes validation renders with `kubectl kustomize` and sends the result to an offline structural/security validator, avoiding ambient cluster contexts. No Make target applies either Terraform or Kubernetes resources.

The EKS node group uses Graviton. `make build-arm64` locally cross-compiles and inspects both production commands; the release workflow must publish a Linux ARM64 or multi-platform image digest before replacing the manifest placeholders.

See [the optional AWS smoke procedure](aws-smoke.md) only after an authorized deployment exists.
