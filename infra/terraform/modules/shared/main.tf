resource "aws_secretsmanager_secret" "api" {
  name                    = "${var.name}/api"
  kms_key_id              = var.kms_key_arn
  recovery_window_in_days = 7
  tags                    = var.tags
}

resource "aws_secretsmanager_secret" "kafka" {
  name                    = "AmazonMSK_${replace(var.name, "-", "_")}"
  kms_key_id              = var.kms_key_arn
  recovery_window_in_days = 7
  tags                    = var.tags
}
