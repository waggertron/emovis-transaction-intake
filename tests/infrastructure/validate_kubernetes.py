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

for document in documents:
    kind = document["kind"]
    if kind != "Deployment":
        continue
    containers = document.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
    if not containers:
        raise SystemExit(f"{document['metadata']['name']} has no containers")
    for container in containers:
        if not container.get("resources", {}).get("requests") or not container.get("resources", {}).get("limits"):
            raise SystemExit(f"{document['metadata']['name']} lacks resource requests or limits")
        image = container.get("image", "")
        if "@sha256:" not in image:
            raise SystemExit(f"{document['metadata']['name']} does not use an immutable image")
