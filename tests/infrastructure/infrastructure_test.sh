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
)
for required in "${required_terraform[@]}"; do
  rg -q "${required}" infra/terraform || { echo "missing Terraform contract: ${required}" >&2; exit 1; }
done
for variable in topic_name topic_partitions topic_retention topic_replication topic_bootstrap_image; do
  rg -q "variable \"${variable}\"" infra/terraform/variables.tf || { echo "missing topic Job variable: ${variable}" >&2; exit 1; }
done
rg -Uq 'resource "kubernetes_job_v1" "topic_bootstrap"[^{]*\{[^}]*count *= *var.local_validation *\? *0 *: *1' infra/terraform/main.tf || { echo "topic Job must be disabled during local validation" >&2; exit 1; }
rg -Uq '(?s)resource "kubernetes_job_v1" "topic_bootstrap".*depends_on *=.*aws_msk_scram_secret_association.main' infra/terraform/main.tf || { echo "topic Job must depend on MSK SCRAM readiness" >&2; exit 1; }
for required in 'storage_encrypted *= *true' 'encrypted *= *true' 'publicly_accessible *= *false' 'endpoint_public_access *= *false' 'iam_database_authentication_enabled *= *true'; do
  rg -q "${required}" infra/terraform || { echo "missing secure Terraform setting: ${required}" >&2; exit 1; }
done
grep -A3 'variable "deletion_protection"' infra/terraform/variables.tf | grep -Fq 'default = true' || { echo "deletion protection must default on" >&2; exit 1; }

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
