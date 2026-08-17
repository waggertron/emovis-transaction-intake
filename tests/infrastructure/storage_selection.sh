#!/usr/bin/env bash
set -euo pipefail

mkdir -p .local/terraform

if make --no-print-directory terraform-plan >/dev/null 2>&1; then
  echo "Terraform planning must fail without an explicit storage selection" >&2
  exit 1
fi

for backend in dynamodb postgres; do
  plan="../../.local/terraform/${backend}-selection.tfplan"
  terraform -chdir=infra/terraform plan -input=false -refresh=false -lock=false \
    -var-file="${backend}.tfvars.example" -out="${plan}" >/dev/null
  rendered="$(terraform -chdir=infra/terraform show -no-color "${plan}")"

  if [[ "${backend}" == "dynamodb" ]]; then
    grep -Fq 'module.dynamodb[0].aws_dynamodb_table.transactions' <<<"${rendered}" || { echo "DynamoDB selection omitted its table" >&2; exit 1; }
    if grep -Fq 'module.postgres[0].aws_db_instance.postgres' <<<"${rendered}"; then
      echo "DynamoDB selection also planned PostgreSQL" >&2
      exit 1
    fi
  else
    grep -Fq 'module.postgres[0].aws_db_instance.postgres' <<<"${rendered}" || { echo "PostgreSQL selection omitted its instance" >&2; exit 1; }
    if grep -Fq 'module.dynamodb[0].aws_dynamodb_table.transactions' <<<"${rendered}"; then
      echo "PostgreSQL selection also planned DynamoDB" >&2
      exit 1
    fi
  fi
done
