# HTTP Adapter Instructions

## Purpose
Expose the transaction application through the documented HTTP API.

## Scope
Applies to the HTTP adapter package.

## Local rules
- Bound request bodies, reject unknown JSON fields, and map errors without leaking internals.
- Authenticate state-changing requests and derive partner identity from credentials.
- Keep handlers limited to transport parsing, application calls, and response mapping.

## Usage
Construct the handler in a composition root with injected application and auth dependencies.

## Validation
Run focused HTTP tests and compare them with the OpenAPI contract.

## Elements
| Element | Behavior |
| --- | --- |
| `handler.go` | Implements bounded authenticated HTTP transport and operational endpoints. |
| `handler_test.go` | Specifies authentication, request parsing, status mapping, and operational endpoints. |
| `static_auth.go` | Authenticates configured API keys using fixed-length constant-time comparisons. |
| `static_auth_test.go` | Specifies secure static API-key authentication behavior. |

## Instruction hierarchy
- Parent: [../AGENTS.md](../AGENTS.md).
- Children: none.
