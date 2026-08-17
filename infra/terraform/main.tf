locals {
  az_count = length(var.availability_zones)
  tags     = merge(var.tags, { Environment = var.environment })
}

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags                 = merge(local.tags, { Name = var.name })
}

data "aws_iam_policy_document" "flow_logs_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["vpc-flow-logs.amazonaws.com"]
    }
  }
}
resource "aws_iam_role" "flow_logs" {
  name               = "${var.name}-vpc-flow-logs"
  assume_role_policy = data.aws_iam_policy_document.flow_logs_assume.json
  tags               = local.tags
}
resource "aws_cloudwatch_log_group" "flow_logs" {
  name              = "/aws/vpc/${var.name}/flow-logs"
  retention_in_days = 30
  kms_key_id        = aws_kms_key.data.arn
  tags              = local.tags
}
data "aws_iam_policy_document" "flow_logs" {
  statement {
    actions = [
      "logs:CreateLogStream",
      "logs:DescribeLogGroups",
      "logs:DescribeLogStreams",
      "logs:PutLogEvents",
    ]
    resources = ["${aws_cloudwatch_log_group.flow_logs.arn}:*"]
  }
}
resource "aws_iam_role_policy" "flow_logs" {
  name   = "${var.name}-vpc-flow-logs"
  role   = aws_iam_role.flow_logs.id
  policy = data.aws_iam_policy_document.flow_logs.json
}
resource "aws_flow_log" "main" {
  iam_role_arn             = aws_iam_role.flow_logs.arn
  log_destination          = aws_cloudwatch_log_group.flow_logs.arn
  traffic_type             = "ALL"
  vpc_id                   = aws_vpc.main.id
  max_aggregation_interval = 60
  tags                     = local.tags
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
  tags   = local.tags
}

resource "aws_subnet" "public" {
  count                   = local.az_count
  vpc_id                  = aws_vpc.main.id
  availability_zone       = var.availability_zones[count.index]
  cidr_block              = cidrsubnet(var.vpc_cidr, 4, count.index)
  map_public_ip_on_launch = false
  tags                    = merge(local.tags, { "kubernetes.io/role/elb" = "1" })
}

resource "aws_subnet" "private" {
  count             = local.az_count
  vpc_id            = aws_vpc.main.id
  availability_zone = var.availability_zones[count.index]
  cidr_block        = cidrsubnet(var.vpc_cidr, 4, count.index + 8)
  tags              = merge(local.tags, { "kubernetes.io/role/internal-elb" = "1" })
}

resource "aws_eip" "nat" {
  count  = local.az_count
  domain = "vpc"
  tags   = local.tags
}
resource "aws_nat_gateway" "main" {
  count         = local.az_count
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id
  depends_on    = [aws_internet_gateway.main]
  tags          = local.tags
}
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  tags   = local.tags
}
resource "aws_route" "internet" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}
resource "aws_route_table_association" "public" {
  count          = local.az_count
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}
resource "aws_route_table" "private" {
  count  = local.az_count
  vpc_id = aws_vpc.main.id
  tags   = local.tags
}
resource "aws_route" "nat" {
  count                  = local.az_count
  route_table_id         = aws_route_table.private[count.index].id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.main[count.index].id
}
resource "aws_route_table_association" "private" {
  count          = local.az_count
  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private[count.index].id
}

resource "aws_kms_key" "data" {
  description             = "${var.name} data encryption"
  enable_key_rotation     = true
  deletion_window_in_days = 30
  tags                    = local.tags
}
resource "aws_kms_alias" "data" {
  name          = "alias/${var.name}-data"
  target_key_id = aws_kms_key.data.key_id
}

data "aws_iam_policy_document" "eks_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["eks.amazonaws.com"]
    }
  }
}
resource "aws_iam_role" "eks" {
  name               = "${var.name}-eks"
  assume_role_policy = data.aws_iam_policy_document.eks_assume.json
  tags               = local.tags
}
resource "aws_iam_role_policy_attachment" "eks_cluster" {
  role       = aws_iam_role.eks.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
}

resource "aws_cloudwatch_log_group" "eks" {
  name              = "/aws/eks/${var.name}/cluster"
  retention_in_days = 30
  kms_key_id        = aws_kms_key.data.arn
  tags              = local.tags
}
resource "aws_eks_cluster" "main" {
  name                      = var.name
  role_arn                  = aws_iam_role.eks.arn
  version                   = var.eks_version
  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]
  vpc_config {
    subnet_ids              = aws_subnet.private[*].id
    endpoint_private_access = true
    endpoint_public_access  = false
  }
  encryption_config {
    provider {
      key_arn = aws_kms_key.data.arn
    }
    resources = ["secrets"]
  }
  depends_on = [aws_iam_role_policy_attachment.eks_cluster, aws_cloudwatch_log_group.eks]
  tags       = local.tags
}

data "tls_certificate" "eks_oidc" {
  url = aws_eks_cluster.main.identity[0].oidc[0].issuer
}
resource "aws_iam_openid_connect_provider" "eks" {
  url             = aws_eks_cluster.main.identity[0].oidc[0].issuer
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.eks_oidc.certificates[0].sha1_fingerprint]
  tags            = local.tags
}

data "aws_iam_policy_document" "nodes_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}
resource "aws_iam_role" "nodes" {
  name               = "${var.name}-nodes"
  assume_role_policy = data.aws_iam_policy_document.nodes_assume.json
  tags               = local.tags
}
resource "aws_iam_role_policy_attachment" "nodes_worker" {
  role       = aws_iam_role.nodes.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}
resource "aws_iam_role_policy_attachment" "nodes_cni" {
  role       = aws_iam_role.nodes.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
}
resource "aws_iam_role_policy_attachment" "nodes_ecr" {
  role       = aws_iam_role.nodes.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}
resource "aws_launch_template" "nodes" {
  name_prefix = "${var.name}-nodes-"
  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 40
      volume_type = "gp3"
      encrypted   = true
      kms_key_id  = aws_kms_key.data.arn
    }
  }
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }
  tag_specifications {
    resource_type = "instance"
    tags          = local.tags
  }
}
resource "aws_eks_node_group" "main" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "system"
  node_role_arn   = aws_iam_role.nodes.arn
  subnet_ids      = aws_subnet.private[*].id
  instance_types  = ["m7g.large"]
  scaling_config {
    desired_size = 2
    min_size     = 2
    max_size     = 6
  }
  update_config { max_unavailable = 1 }
  launch_template {
    id      = aws_launch_template.nodes.id
    version = aws_launch_template.nodes.latest_version
  }
  depends_on = [aws_iam_role_policy_attachment.nodes_worker, aws_iam_role_policy_attachment.nodes_cni, aws_iam_role_policy_attachment.nodes_ecr]
  tags       = local.tags
}

resource "aws_security_group" "data" {
  name        = "${var.name}-data"
  description = "EKS access to private data services"
  vpc_id      = aws_vpc.main.id
  tags        = local.tags
}
resource "aws_security_group_rule" "data_egress" {
  type              = "egress"
  security_group_id = aws_security_group.data.id
  protocol          = "-1"
  from_port         = 0
  to_port           = 0
  cidr_blocks       = [var.vpc_cidr]
}

resource "aws_cloudwatch_log_group" "msk" {
  name              = "/aws/msk/${var.name}"
  retention_in_days = 30
  kms_key_id        = aws_kms_key.data.arn
  tags              = local.tags
}
resource "aws_msk_configuration" "main" {
  name              = var.name
  kafka_versions    = [var.kafka_version]
  server_properties = <<-EOF
    auto.create.topics.enable=false
    default.replication.factor=3
    min.insync.replicas=2
    num.partitions=3
  EOF
}
resource "aws_msk_cluster" "main" {
  cluster_name           = var.name
  kafka_version          = var.kafka_version
  number_of_broker_nodes = local.az_count
  broker_node_group_info {
    instance_type   = var.msk_instance_type
    client_subnets  = aws_subnet.private[*].id
    security_groups = [aws_security_group.data.id]
    storage_info {
      ebs_storage_info { volume_size = 100 }
    }
  }
  client_authentication {
    sasl { scram = true }
    unauthenticated = false
  }
  encryption_info {
    encryption_at_rest_kms_key_arn = aws_kms_key.data.arn
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }
  configuration_info {
    arn      = aws_msk_configuration.main.arn
    revision = aws_msk_configuration.main.latest_revision
  }
  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.msk.name
      }
    }
  }
  tags = local.tags
}
resource "aws_msk_scram_secret_association" "main" {
  count           = var.runtime_secrets_ready ? 1 : 0
  cluster_arn     = aws_msk_cluster.main.arn
  secret_arn_list = [module.shared.kafka_secret_arn]
}

resource "kubernetes_namespace_v1" "transaction_intake" {
  count = var.local_validation ? 0 : 1
  metadata { name = "transaction-intake" }
}

resource "kubernetes_service_account_v1" "transaction_intake" {
  count = var.local_validation ? 0 : 1
  metadata {
    name      = "transaction-intake"
    namespace = kubernetes_namespace_v1.transaction_intake[0].metadata[0].name
    annotations = {
      "eks.amazonaws.com/role-arn" = aws_iam_role.workload.arn
    }
  }
}

resource "kubernetes_job_v1" "topic_bootstrap" {
  count = !var.local_validation && var.runtime_secrets_ready ? 1 : 0
  metadata {
    name      = "transaction-intake-topic-bootstrap"
    namespace = kubernetes_namespace_v1.transaction_intake[0].metadata[0].name
  }
  spec {
    backoff_limit              = 6
    ttl_seconds_after_finished = 3600
    template {
      metadata {
        labels = { app = "transaction-intake-topic-bootstrap" }
      }
      spec {
        service_account_name = kubernetes_service_account_v1.transaction_intake[0].metadata[0].name
        restart_policy       = "OnFailure"
        security_context {
          run_as_non_root = true
          seccomp_profile {
            type = "RuntimeDefault"
          }
        }
        container {
          name  = "topic-bootstrap"
          image = var.topic_bootstrap_image
          env {
            name  = "KAFKA_BROKERS"
            value = aws_msk_cluster.main.bootstrap_brokers_sasl_scram
          }
          env {
            name  = "KAFKA_TOPIC"
            value = var.topic_name
          }
          env {
            name  = "KAFKA_TOPIC_PARTITIONS"
            value = tostring(var.topic_partitions)
          }
          env {
            name  = "KAFKA_TOPIC_REPLICATION"
            value = tostring(var.topic_replication)
          }
          env {
            name  = "KAFKA_TOPIC_RETENTION"
            value = var.topic_retention
          }
          env {
            name  = "KAFKA_TLS"
            value = "true"
          }
          env {
            name  = "AWS_SECRET_ID"
            value = module.shared.api_secret_name
          }
          resources {
            requests = { cpu = "50m", memory = "64Mi" }
            limits   = { cpu = "250m", memory = "128Mi" }
          }
          security_context {
            allow_privilege_escalation = false
            read_only_root_filesystem  = true
            run_as_non_root            = true
            capabilities {
              drop = ["ALL"]
            }
          }
        }
      }
    }
  }
  depends_on = [aws_msk_scram_secret_association.main, kubernetes_service_account_v1.transaction_intake]
}

module "shared" {
  source      = "./modules/shared"
  name        = var.name
  kms_key_arn = aws_kms_key.data.arn
  tags        = local.tags
}

module "dynamodb" {
  count               = var.storage_backend == "dynamodb" ? 1 : 0
  source              = "./modules/dynamodb"
  name                = var.name
  kms_key_arn         = aws_kms_key.data.arn
  deletion_protection = var.deletion_protection
  tags                = local.tags
}

module "postgres" {
  count               = var.storage_backend == "postgres" ? 1 : 0
  source              = "./modules/postgres"
  name                = var.name
  kms_key_arn         = aws_kms_key.data.arn
  subnet_ids          = aws_subnet.private[*].id
  security_group_id   = aws_security_group.data.id
  instance_class      = var.db_instance_class
  deletion_protection = var.deletion_protection
  tags                = local.tags
}

data "aws_iam_policy_document" "workload" {
  dynamic "statement" {
    for_each = var.storage_backend == "dynamodb" ? [1] : []
    content {
      sid       = "DynamoTransactions"
      actions   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:TransactWriteItems", "dynamodb:DescribeTable"]
      resources = [module.dynamodb[0].table_arn, module.dynamodb[0].index_arn]
    }
  }
  statement {
    sid       = "ReadRuntimeSecrets"
    actions   = ["secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"]
    resources = concat([module.shared.api_secret_arn, module.shared.kafka_secret_arn], module.postgres[*].secret_arn)
  }
  statement {
    sid       = "UseDataKey"
    actions   = ["kms:Decrypt"]
    resources = [aws_kms_key.data.arn]
  }
}
data "aws_iam_policy_document" "workload_assume" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.eks.arn]
    }
    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "${replace(aws_iam_openid_connect_provider.eks.url, "https://", "")}:sub"
      values   = ["system:serviceaccount:transaction-intake:transaction-intake"]
    }
  }
}
resource "aws_iam_policy" "workload" {
  name   = "${var.name}-workload"
  policy = data.aws_iam_policy_document.workload.json
  tags   = local.tags
}
resource "aws_iam_role" "workload" {
  name               = "${var.name}-workload"
  assume_role_policy = data.aws_iam_policy_document.workload_assume.json
  tags               = local.tags
}
resource "aws_iam_role_policy_attachment" "workload" {
  role       = aws_iam_role.workload.name
  policy_arn = aws_iam_policy.workload.arn
}

resource "aws_cloudwatch_metric_alarm" "msk_cpu" {
  alarm_name          = "${var.name}-msk-cpu"
  namespace           = "AWS/Kafka"
  metric_name         = "CpuUser"
  statistic           = "Average"
  period              = 300
  evaluation_periods  = 3
  threshold           = 70
  comparison_operator = "GreaterThanThreshold"
  dimensions          = { "Cluster Name" = aws_msk_cluster.main.cluster_name }
  alarm_description   = "Sustained MSK broker CPU requires investigation"
  tags                = local.tags
}
