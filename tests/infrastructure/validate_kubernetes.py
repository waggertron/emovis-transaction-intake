#!/usr/bin/env python3
import sys

import yaml


documents = list(yaml.safe_load_all(sys.stdin))
if not documents:
    raise SystemExit("no Kubernetes documents rendered")

identities = set()
for document in documents:
    if not isinstance(document, dict):
        raise SystemExit("Kubernetes document must be a mapping")
    api_version = document.get("apiVersion")
    kind = document.get("kind")
    metadata = document.get("metadata")
    if not isinstance(api_version, str) or not isinstance(kind, str):
        raise SystemExit("Kubernetes document lacks apiVersion or kind")
    if not isinstance(metadata, dict) or not isinstance(metadata.get("name"), str):
        raise SystemExit(f"{kind} lacks metadata.name")
    identity = (kind, metadata["name"])
    if identity in identities:
        raise SystemExit(f"duplicate Kubernetes object: {identity}")
    identities.add(identity)

required = {
    ("Namespace", "transaction-intake"),
    ("Deployment", "transaction-intake-api"),
    ("Deployment", "transaction-intake-worker"),
    ("Job", "transaction-intake-topic-bootstrap"),
    ("ServiceAccount", "transaction-intake"),
    ("HorizontalPodAutoscaler", "transaction-intake-api"),
    ("HorizontalPodAutoscaler", "transaction-intake-worker"),
    ("PodDisruptionBudget", "transaction-intake-api"),
    ("PodDisruptionBudget", "transaction-intake-worker"),
}
missing = required - identities
if missing:
    raise SystemExit(f"missing Kubernetes objects: {sorted(missing)}")

service_accounts = {name for kind, name in identities if kind == "ServiceAccount"}
secrets = {name for kind, name in identities if kind == "Secret"}

for document in documents:
    kind = document["kind"]
    if kind not in {"Deployment", "Job"}:
        continue
    pod_spec = document.get("spec", {}).get("template", {}).get("spec", {})
    account = pod_spec.get("serviceAccountName")
    if account and account not in service_accounts:
        raise SystemExit(f"{document['metadata']['name']} references missing ServiceAccount {account}")
    containers = pod_spec.get("containers", [])
    if not containers:
        raise SystemExit(f"{document['metadata']['name']} has no containers")
    for container in containers:
        if not container.get("resources", {}).get("requests") or not container.get("resources", {}).get("limits"):
            raise SystemExit(f"{document['metadata']['name']} lacks resource requests or limits")
        image = container.get("image", "")
        if "@sha256:" not in image:
            raise SystemExit(f"{document['metadata']['name']} does not use an immutable image")
        for entry in container.get("env", []):
            secret_ref = entry.get("valueFrom", {}).get("secretKeyRef", {})
            if secret_ref and secret_ref.get("name") not in secrets:
                raise SystemExit(f"{document['metadata']['name']} references missing Secret {secret_ref.get('name')}")

    env_names = {entry.get("name") for container in containers for entry in container.get("env", [])}
    required_env = {"AWS_SECRET_ID"}
    if document["metadata"]["name"] != "transaction-intake-api":
        required_env.add("KAFKA_TLS")
    missing_env = required_env - env_names
    if missing_env:
        raise SystemExit(f"{document['metadata']['name']} lacks required environment: {sorted(missing_env)}")

    if kind == "Deployment":
        selector = document.get("spec", {}).get("selector", {}).get("matchLabels", {})
        labels = document.get("spec", {}).get("template", {}).get("metadata", {}).get("labels", {})
        if not selector or any(labels.get(key) != value for key, value in selector.items()):
            raise SystemExit(f"{document['metadata']['name']} selector does not match pod labels")
