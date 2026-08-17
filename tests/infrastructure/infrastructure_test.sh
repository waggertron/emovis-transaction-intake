#!/usr/bin/env bash
set -euo pipefail

required_terraform=(
  'resource "aws_vpc"' 'resource "aws_eks_cluster"' 'resource "aws_msk_cluster"'
  'resource "aws_dynamodb_table"' 'resource "aws_db_instance"'
  'resource "aws_secretsmanager_secret"' 'resource "aws_iam_role"'
  'resource "aws_iam_openid_connect_provider"' 'resource "aws_msk_scram_secret_association"'
  'resource "aws_cloudwatch_log_group"' 'resource "aws_cloudwatch_metric_alarm"'
  'resource "aws_flow_log"'
  'resource "kubernetes_job_v1" "topic_bootstrap"'
	  'resource "kubernetes_namespace_v1" "transaction_intake"'
	  'resource "kubernetes_service_account_v1" "transaction_intake"'
)
for required in "${required_terraform[@]}"; do
  rg -q "${required}" infra/terraform || { echo "missing Terraform contract: ${required}" >&2; exit 1; }
done
for module in shared dynamodb postgres; do
  [[ -f "infra/terraform/modules/${module}/main.tf" ]] || { echo "missing Terraform module: ${module}" >&2; exit 1; }
done
grep -A8 'variable "storage_backend"' infra/terraform/variables.tf | grep -Fq 'contains(["dynamodb", "postgres"], var.storage_backend)' || {
  echo "storage_backend must require an explicit supported selection" >&2
  exit 1
}
if grep -A8 'variable "storage_backend"' infra/terraform/variables.tf | grep -Eq '^[[:space:]]*default[[:space:]]*='; then
  echo "storage_backend must not have a default" >&2
  exit 1
fi
for backend in dynamodb postgres; do
  [[ -f "infra/terraform/${backend}.tfvars.example" ]] || { echo "missing explicit ${backend} example selection" >&2; exit 1; }
  grep -Eq "^storage_backend[[:space:]]*=[[:space:]]*\"${backend}\"$" "infra/terraform/${backend}.tfvars.example" || { echo "${backend} example does not select its backend" >&2; exit 1; }
done
if rg -q '^resource "aws_(dynamodb_table|db_instance|db_subnet_group)"' infra/terraform/main.tf; then
  echo "storage resources must be owned by selectable modules" >&2
  exit 1
fi
grep -Eq 'count[[:space:]]*=[[:space:]]*var.storage_backend == "dynamodb" \? 1 : 0' infra/terraform/main.tf || { echo "DynamoDB module is not explicitly selected" >&2; exit 1; }
grep -Eq 'count[[:space:]]*=[[:space:]]*var.storage_backend == "postgres" \? 1 : 0' infra/terraform/main.tf || { echo "PostgreSQL module is not explicitly selected" >&2; exit 1; }
grep -Fq 'name: AWS_SECRET_ID' deploy/kubernetes/topic-bootstrap-job.yaml || { echo "topic bootstrap must load credentials through AWS Secrets Manager" >&2; exit 1; }
if grep -Fq 'secretKeyRef:' deploy/kubernetes/topic-bootstrap-job.yaml; then
  echo "topic bootstrap has a dangling Kubernetes Secret dependency" >&2
  exit 1
fi
rg -Uq '(?s)resource "kubernetes_job_v1" "topic_bootstrap".*name *= *"AWS_SECRET_ID"' infra/terraform/main.tf || { echo "Terraform topic bootstrap must select the AWS secret provider" >&2; exit 1; }
if rg -Uq '(?s)resource "kubernetes_job_v1" "topic_bootstrap".*secret_key_ref' infra/terraform/main.tf; then
  echo "Terraform topic bootstrap has a dangling Kubernetes Secret dependency" >&2
  exit 1
fi
for variable in topic_name topic_partitions topic_retention topic_replication topic_bootstrap_image; do
  rg -q "variable \"${variable}\"" infra/terraform/variables.tf || { echo "missing topic Job variable: ${variable}" >&2; exit 1; }
done
rg -Uq 'resource "kubernetes_job_v1" "topic_bootstrap"[^{]*\{[^}]*count *= *!var.local_validation *&& *var.runtime_secrets_ready *\? *1 *: *0' infra/terraform/main.tf || { echo "topic Job must require cloud mode and populated runtime secrets" >&2; exit 1; }
rg -Uq 'resource "aws_msk_scram_secret_association" "main"[^{]*\{[^}]*count *= *var.runtime_secrets_ready *\? *1 *: *0' infra/terraform/main.tf || { echo "MSK SCRAM association must wait for explicit secret population" >&2; exit 1; }
rg -Uq '(?s)resource "kubernetes_job_v1" "topic_bootstrap".*depends_on *=.*aws_msk_scram_secret_association.main' infra/terraform/main.tf || { echo "topic Job must depend on MSK SCRAM readiness" >&2; exit 1; }
for required in 'storage_encrypted *= *true' 'encrypted *= *true' 'publicly_accessible *= *false' 'endpoint_public_access *= *false' 'iam_database_authentication_enabled *= *true'; do
  rg -q "${required}" infra/terraform || { echo "missing secure Terraform setting: ${required}" >&2; exit 1; }
done
grep -A3 'variable "deletion_protection"' infra/terraform/variables.tf | grep -Fq 'default = true' || { echo "deletion protection must default on" >&2; exit 1; }
if rg -q '^    (hash_key|range_key)[[:space:]]*=' infra/terraform -g '*.tf'; then
  echo "DynamoDB indexes must not use deprecated hash_key/range_key arguments" >&2
  exit 1
fi
for key in 'attribute_name[[:space:]]*=[[:space:]]*"dispatch_pk"' 'key_type[[:space:]]*=[[:space:]]*"HASH"' 'attribute_name[[:space:]]*=[[:space:]]*"dispatch_sk"' 'key_type[[:space:]]*=[[:space:]]*"RANGE"'; do
  rg -q "${key}" infra/terraform -g '*.tf' || { echo "missing current DynamoDB index key schema: ${key}" >&2; exit 1; }
done

for manifest in deploy/kubernetes/api.yaml deploy/kubernetes/worker.yaml deploy/kubernetes/topic-bootstrap-job.yaml; do
  [[ -f "${manifest}" ]] || { echo "missing Kubernetes manifest: ${manifest}" >&2; exit 1; }
  grep -Fq 'runAsNonRoot: true' "${manifest}" || { echo "non-root policy missing: ${manifest}" >&2; exit 1; }
  grep -Fq 'allowPrivilegeEscalation: false' "${manifest}" || { echo "privilege policy missing: ${manifest}" >&2; exit 1; }
  grep -Fq 'resources:' "${manifest}" || { echo "resource bounds missing: ${manifest}" >&2; exit 1; }
  grep -Eq 'image: .+@sha256:[a-f0-9]{64}' "${manifest}" || { echo "immutable image missing: ${manifest}" >&2; exit 1; }
done
for manifest in deploy/kubernetes/api-hpa.yaml deploy/kubernetes/worker-hpa.yaml deploy/kubernetes/disruption-budgets.yaml; do
  [[ -f "${manifest}" ]] || { echo "missing Kubernetes availability manifest: ${manifest}" >&2; exit 1; }
done
for hpa in deploy/kubernetes/api-hpa.yaml deploy/kubernetes/worker-hpa.yaml; do
  grep -Fq 'minReplicas:' "${hpa}" || { echo "minimum replicas missing: ${hpa}" >&2; exit 1; }
  grep -Fq 'maxReplicas:' "${hpa}" || { echo "maximum replicas missing: ${hpa}" >&2; exit 1; }
  grep -Fq 'averageUtilization:' "${hpa}" || { echo "resource utilization target missing: ${hpa}" >&2; exit 1; }
done
grep -Fq 'kind: PodDisruptionBudget' deploy/kubernetes/disruption-budgets.yaml || { echo "pod disruption budgets missing" >&2; exit 1; }
grep -Fq 'readinessProbe:' deploy/kubernetes/api.yaml || { echo "API readiness probe missing" >&2; exit 1; }
grep -Fq 'livenessProbe:' deploy/kubernetes/api.yaml || { echo "API liveness probe missing" >&2; exit 1; }
for manifest in deploy/kubernetes/api.yaml deploy/kubernetes/worker.yaml; do
  grep -Fq 'name: AWS_SECRET_ID' "${manifest}" || { echo "AWS Secrets Manager identifier missing: ${manifest}" >&2; exit 1; }
  if grep -Fq 'secretKeyRef:' "${manifest}"; then
    echo "application workloads must use the IRSA-backed provider directly: ${manifest}" >&2
    exit 1
  fi
done
if rg -n '(password|api.?key):[[:space:]]+[^$<{]' deploy/kubernetes infra/terraform; then
  echo "possible plaintext secret in infrastructure" >&2
  exit 1
fi
