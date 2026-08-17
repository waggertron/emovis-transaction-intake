output "eks_cluster_name" {
  value = aws_eks_cluster.main.name
}
output "msk_bootstrap_brokers_sasl_scram" {
  value     = aws_msk_cluster.main.bootstrap_brokers_sasl_scram
  sensitive = true
}
output "dynamodb_table_name" {
  value = aws_dynamodb_table.transactions.name
}
output "postgres_endpoint" {
  value     = aws_db_instance.postgres.endpoint
  sensitive = true
}
output "workload_role_arn" {
  value = aws_iam_role.workload.arn
}
output "runtime_secret_arns" {
  value = [aws_secretsmanager_secret.api.arn, aws_secretsmanager_secret.postgres.arn, aws_secretsmanager_secret.kafka.arn]
}
