output "api_secret_arn" {
  value = aws_secretsmanager_secret.api.arn
}

output "api_secret_name" {
  value = aws_secretsmanager_secret.api.name
}

output "kafka_secret_arn" {
  value = aws_secretsmanager_secret.kafka.arn
}
