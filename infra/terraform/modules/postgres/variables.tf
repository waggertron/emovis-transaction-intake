variable "name" {
  type = string
}

variable "kms_key_arn" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "security_group_id" {
  type = string
}

variable "instance_class" {
  type = string
}

variable "deletion_protection" {
  type = bool
}

variable "tags" {
  type = map(string)
}
