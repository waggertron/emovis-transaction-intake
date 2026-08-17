# Optional non-production AWS smoke procedure

This procedure is supplemental evidence and is never required to complete local validation. It must run only after a human-authorized deployment in an explicitly selected non-production account.

1. Confirm the caller identity and region through the organization's approved AWS authentication workflow. Stop unless the account is designated non-production.
2. Review the saved Terraform plan, expected cost, deletion protection, remote state, image digests, networking, and secret ownership with a second operator. This repository intentionally exposes no apply target.
3. Confirm EKS has a private endpoint and the API/worker service account resolves to the expected IRSA role.
4. Confirm MSK reports TLS-only client traffic, SASL/SCRAM enabled, encryption at rest, broker logging, and the expected topic partition/retention settings.
5. Populate Secrets Manager through the approved secret-management workflow. Never place values in shell history, Terraform variables, manifests, logs, or screenshots.
6. From an approved network path, submit a unique README-shaped transaction through the deployed ingress. Verify `201`, identical replay `200` with `Idempotent-Replay: true`, and changed replay `409`.
7. Observe exactly one stable event ID at the authorized Kafka consumer, verify the key is `partnerId:transactionId`, and confirm the envelope contains no credentials.
8. Verify the selected DynamoDB or PostgreSQL store has one transaction and one published outbox record. Review API/worker logs, metrics, outbox backlog, MSK alarm state, and consumer lag for unexpected errors or secret/PII exposure.
9. Record account ID, region, deployment revision, test transaction ID, UTC time, operator, and sanitized evidence in the organization's audit system—not in this repository.
10. Remove only the unique smoke transaction if the approved retention process permits it. Infrastructure teardown requires a separate reviewed plan and explicit authorization.
