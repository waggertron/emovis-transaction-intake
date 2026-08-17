# Secret Configuration Instructions

## Purpose
Load external configuration values without embedding or logging secret material.

## Scope
Applies to secret-provider ports and adapters in this package.

## Local rules
- Bound input size and reject malformed or empty secret values.
- Reload provider data on each explicit load so rotation can be adopted on process restart or refresh.
- Wrap failures with provider context but never include secret contents.

## Usage
Select the local JSON file with `LOCAL_SECRET_FILE`; cloud composition uses the same provider contract.

## Validation
Run unit and component tests for lookup, malformed data, rotation-shaped reload, unavailability, and redaction.

## Elements
| Element | Behavior |
| --- | --- |
| `aws.go` | Loads bounded JSON configuration through the AWS Secrets Manager client boundary. |
| `aws_test.go` | Proves cloud lookup, decoding, classification, cancellation, and redaction with a deterministic fake. |
| `file.go` | Implements the bounded local JSON secret provider and environment overlay. |
| `file_test.go` | Specifies lookup, reload, rejection, and redacted failures. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
