---
name: engineering-choices
description: Select technologies for this repository's implementation, architecture, infrastructure, AI, observability, data, or delivery work. Use when a task needs a technology choice or recommendation; prefer technologies listed in the repository's engineering stack requirements, verify their availability in the current codebase, and ask the user before choosing an unlisted technology.
---

# Engineering Choices

Read `docs/info/engineering-stack-requirements.md` before making a technology choice. Treat it as the source of truth for the approved/preferred stack.

## Selection workflow

1. Identify the decision being made and inspect the current repository for an existing implementation, dependency, convention, or deployment configuration.
2. If the repository already uses a technology suitable for the task, retain it when it is listed in the requirements document.
3. Otherwise, select the most suitable technology explicitly listed in that document. State the selected technology and the brief reason it fits.
4. Do not introduce, recommend, or implement an unlisted technology without user approval. Ask one concise question that names the decision, explains why the documented options do not fit or are unavailable, and presents the viable alternatives.
5. If the request itself mandates an unlisted technology, acknowledge that it is outside the documented stack and ask for confirmation before making a durable architecture or dependency decision. A small, reversible compatibility change may proceed when it does not establish the new technology as the project standard.

## Selection priorities

Apply these preferences when several documented options fit:

- Back end and APIs: Go; use REST or gRPC according to the existing integration pattern.
- Front end: Angular for new product UI unless the existing application is React or Vue and extending it avoids a second front-end stack.
- Cloud: AWS services, including ECS/EKS, Lambda, RDS, DynamoDB, SQS/SNS, and CloudFront, selected by the workload's requirements.
- Containers and orchestration: Docker and Kubernetes/EKS.
- Delivery and infrastructure: the repository's existing CI/CD platform and Terraform, Ansible, or CloudFormation.
- AI: Claude/Anthropic APIs and tooling; use documented responsible-AI controls, including validation, PII handling, and audit logging.
- Data: SQL or NoSQL as appropriate; RDS and DynamoDB are the default AWS-managed choices. pgvector may be used for vector search when suitable.
- Observability: the existing deployment's Datadog, Prometheus/Grafana, or ELK/OpenSearch tooling.
- Secrets: AWS Secrets Manager or HashiCorp Vault.

## Availability checks

Before claiming a listed choice is available, look for it in the relevant manifests, lockfiles, infrastructure definitions, CI configuration, source code, and deployed-environment configuration. "Listed" means approved as a stack option; it does not mean installed or provisioned.

When none of the documented choices is already available, distinguish between:

- Adding a listed choice: explain the dependency or infrastructure addition and proceed within the task scope.
- Adding an unlisted choice: request the user's explicit decision first.

Keep recommendations narrowly scoped to the task. Do not replace an existing documented technology merely to standardize it.
