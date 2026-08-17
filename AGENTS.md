# Repository Instructions

## Purpose
Define repository-wide rules for the transaction intake service.

## Scope
Applies everywhere unless a nearer `AGENTS.md` is stricter.

## Local rules
- Write and run focused failing tests before production code.
- Record material decisions in the current plan.
- Keep the core independent of transports and infrastructure.
- Use Make targets for supported workflows and never commit secrets.

## Usage
Read this file and the nearest child instructions before changing files.

## Validation
Run the hierarchy validator, focused tests, and repository validation target.

## Elements
| Element | Behavior |
| --- | --- |
| `.codex` | Contains repository-local Codex skills. |
| `.dockerignore` | Excludes VCS, local build, test, and runtime artifacts from image contexts. |
| `.env.local.example` | Documents a safe, runnable local configuration without real credentials. |
| `.env.production.example` | Documents the full production-shaped configuration with non-secret placeholders. |
| `.gitignore` | Excludes local build, test, runtime, and all non-example dotenv artifacts. |
| `.gitleaks.toml` | Allows only named local sentinel keys while scanning source and history. |
| `.github` | Contains least-privilege hosted CI configuration. |
| `Dockerfile` | Builds reproducible non-root images for each executable mode. |
| `Makefile` | Provides the canonical strict developer and CI command interface. |
| `README.md` | Provides onboarding, operation, and extensive architecture documentation. |
| `api` | Contains the explicitly mocked OpenAPI contract. |
| `cmd` | Contains executable composition roots. |
| `compose.yaml` | Runs the complete local Kafka and transaction-intake system. |
| `docs` | Contains plans, requirements, and architecture. |
| `go.mod` | Declares the Go module and supported language version. |
| `go.sum` | Locks authenticated checksums for Go dependencies. |
| `internal` | Contains non-exported domain, application, adapter, and bootstrap code. |
| `infra` | Contains locally validated cloud infrastructure definitions. |
| `deploy` | Contains immutable, hardened deployment manifests. |
| `tests` | Contains repository-level command, contract, integration, and smoke checks. |

## Instruction hierarchy
- Parent: none.
- Child: [.codex/AGENTS.md](.codex/AGENTS.md).
- Child: [.github/AGENTS.md](.github/AGENTS.md).
- Child: [api/AGENTS.md](api/AGENTS.md).
- Child: [cmd/AGENTS.md](cmd/AGENTS.md).
- Child: [docs/AGENTS.md](docs/AGENTS.md).
- Child: [internal/AGENTS.md](internal/AGENTS.md).
- Child: [infra/AGENTS.md](infra/AGENTS.md).
- Child: [deploy/AGENTS.md](deploy/AGENTS.md).
- Child: [tests/AGENTS.md](tests/AGENTS.md).
