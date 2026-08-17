resource "aws_db_subnet_group" "main" {
  name       = var.name
  subnet_ids = var.subnet_ids
  tags       = var.tags
}

resource "aws_secretsmanager_secret" "postgres" {
  name                    = "${var.name}/postgres"
  kms_key_id              = var.kms_key_arn
  recovery_window_in_days = 7
  tags                    = var.tags
}

resource "aws_db_instance" "postgres" {
  identifier                          = var.name
  engine                              = "postgres"
  engine_version                      = "17.6"
  instance_class                      = var.instance_class
  allocated_storage                   = 30
  max_allocated_storage               = 100
  storage_type                        = "gp3"
  storage_encrypted                   = true
  kms_key_id                          = var.kms_key_arn
  db_name                             = "transactions"
  username                            = "transaction_admin"
  manage_master_user_password         = true
  iam_database_authentication_enabled = true
  master_user_secret_kms_key_id       = var.kms_key_arn
  db_subnet_group_name                = aws_db_subnet_group.main.name
  vpc_security_group_ids              = [var.security_group_id]
  publicly_accessible                 = false
  multi_az                            = true
  backup_retention_period             = 7
  deletion_protection                 = var.deletion_protection
  skip_final_snapshot                 = !var.deletion_protection
  enabled_cloudwatch_logs_exports     = ["postgresql", "upgrade"]
  tags                                = var.tags
}
