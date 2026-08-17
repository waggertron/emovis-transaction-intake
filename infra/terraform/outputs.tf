output "eks_cluster_name" {
  value = aws_eks_cluster.main.name
}
output "msk_bootstrap_brokers_sasl_scram" {
  value     = aws_msk_cluster.main.bootstrap_brokers_sasl_scram
  sensitive = true
}
output "dynamodb_table_name" {
  value = try(module.dynamodb[0].table_name, null)
}
output "postgres_endpoint" {
  value     = try(module.postgres[0].endpoint, null)
  sensitive = true
}
output "workload_role_arn" {
  value = aws_iam_role.workload.arn
}
output "runtime_secret_arns" {
  value = concat([module.shared.api_secret_arn, module.shared.kafka_secret_arn], module.postgres[*].secret_arn)
}
