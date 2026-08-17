# Bootstrap Instructions

## Purpose
Load process configuration and construct secure runtime boundaries.

## Scope
Applies to the bootstrap package.

## Local rules
- Keep secrets externalized and never log configuration values containing them.
- Configure HTTP timeouts and limits explicitly.
- Keep business decisions out of composition code.

## Usage
Use from command entry points to create process dependencies.

## Validation
Run focused bootstrap tests and static analysis.

## Elements
| Element | Behavior |
| --- | --- |
| `config.go` | Loads required secrets, identity, address, and process mode. |
| `id.go` | Generates cryptographically random RFC 4122 UUIDs. |
| `server.go` | Constructs a resource-bounded HTTP server. |
| `secrets` | Loads external configuration values through local and cloud provider boundaries. |
| `bootstrap_test.go` | Specifies configuration, mode, server, and secure-ID behavior. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Child: [secrets/AGENTS.md](secrets/AGENTS.md).
