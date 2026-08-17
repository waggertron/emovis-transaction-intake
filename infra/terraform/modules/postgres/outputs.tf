output "endpoint" {
  value     = aws_db_instance.postgres.endpoint
  sensitive = true
}

output "secret_arn" {
  value = aws_secretsmanager_secret.postgres.arn
}
