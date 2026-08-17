resource "aws_dynamodb_table" "transactions" {
  name         = var.name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"
  range_key    = "sk"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }
  attribute {
    name = "dispatch_pk"
    type = "S"
  }
  attribute {
    name = "dispatch_sk"
    type = "S"
  }

  global_secondary_index {
    name = "outbox-dispatch"
    key_schema {
      attribute_name = "dispatch_pk"
      key_type       = "HASH"
    }
    key_schema {
      attribute_name = "dispatch_sk"
      key_type       = "RANGE"
    }
    projection_type = "ALL"
  }

  point_in_time_recovery { enabled = true }
  server_side_encryption {
    enabled     = true
    kms_key_arn = var.kms_key_arn
  }
  deletion_protection_enabled = var.deletion_protection
  tags                        = var.tags
}
