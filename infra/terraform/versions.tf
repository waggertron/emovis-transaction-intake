terraform {
  required_version = ">= 1.8.0, < 2.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.38"
    }
  }
}

provider "aws" {
  region                      = var.aws_region
  access_key                  = var.local_validation ? "mock-access-key" : null
  secret_key                  = var.local_validation ? "mock-secret-key" : null
  skip_credentials_validation = var.local_validation
  skip_requesting_account_id  = var.local_validation
  skip_metadata_api_check     = var.local_validation
  skip_region_validation      = var.local_validation
}

data "aws_eks_cluster_auth" "main" {
  count = var.local_validation ? 0 : 1
  name  = aws_eks_cluster.main.name
}

provider "kubernetes" {
  host                   = var.local_validation ? "https://127.0.0.1" : aws_eks_cluster.main.endpoint
  cluster_ca_certificate = var.local_validation ? "" : base64decode(aws_eks_cluster.main.certificate_authority[0].data)
  token                  = var.local_validation ? "local-validation-only" : data.aws_eks_cluster_auth.main[0].token
}
