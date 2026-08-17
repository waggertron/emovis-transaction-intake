variable "aws_region" {
  type    = string
  default = "us-west-2"
}
variable "availability_zones" {
  type    = list(string)
  default = ["us-west-2a", "us-west-2b", "us-west-2c"]
}
variable "name" {
  type    = string
  default = "emovis-transaction-intake"
}
variable "vpc_cidr" {
  type    = string
  default = "10.42.0.0/16"
}
variable "local_validation" {
  type    = bool
  default = true
}
variable "storage_backend" {
  type        = string
  description = "Explicit persistence selection; no backend is selected implicitly"

  validation {
    condition     = contains(["dynamodb", "postgres"], var.storage_backend)
    error_message = "storage_backend must explicitly be dynamodb or postgres"
  }
}
variable "runtime_secrets_ready" {
  type        = bool
  default     = false
  description = "Operator confirmation that externally populated API and MSK SCRAM secret values are ready for association and topic bootstrap"
}
variable "environment" {
  type    = string
  default = "nonprod"
}
variable "eks_version" {
  type    = string
  default = "1.33"
}
variable "kafka_version" {
  type    = string
  default = "3.8.x"
}
variable "db_instance_class" {
  type    = string
  default = "db.t4g.small"
}
variable "msk_instance_type" {
  type    = string
  default = "kafka.t3.small"
}
variable "deletion_protection" {
  type    = bool
  default = true
}

variable "topic_name" {
  type    = string
  default = "transaction.review-candidates.v1"
}
variable "topic_partitions" {
  type    = number
  default = 3
}
variable "topic_replication" {
  type    = number
  default = 3
}
variable "topic_retention" {
  type    = string
  default = "168h"
}
variable "topic_bootstrap_image" {
  type        = string
  description = "Immutable topic-bootstrap image reference"
  default     = "public.ecr.aws/example/topic-bootstrap@sha256:0000000000000000000000000000000000000000000000000000000000000000"
  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.topic_bootstrap_image))
    error_message = "topic_bootstrap_image must use an immutable sha256 digest"
  }
}

variable "tags" {
  type = map(string)
  default = {
    Project   = "emovis-transaction-intake"
    ManagedBy = "terraform"
  }
}
