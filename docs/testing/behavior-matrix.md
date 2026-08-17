# Behavior-to-test matrix

Statement coverage is a floor. The named evidence below is the correctness argument; component and E2E tests add real wiring confidence and do not replace unit assertions.

| Critical behavior | Focused unit/contract evidence | Real component/E2E evidence |
| --- | --- | --- |
| Valid and invalid toll fields, positive minor-unit billable amount, supported currency, UTC time, identifiers | `TestTransactionValidateAcceptsValidFixture`, `TestTransactionValidateRejectsInvalidFields`, `TestTransactionFingerprintRejectsInvalidTransaction` | Every `make test-e2e-*` submits the contract-valid billable object; contract tests validate OpenAPI examples. |
| Canonical identity and fingerprint | `TestTransactionFingerprintIsCanonicalAndSensitive` | Memory, NDJSON, PostgreSQL, and DynamoDB E2E replay/conflict assertions. |
| Atomic transaction plus outbox acceptance | `TestIntakeAcceptsTransactionWithPendingOutboxEvent`, adapter `TestStoreAccepts*` and rollback tests | Shared store contract under `make test-component-storage`; separate-process PostgreSQL/DynamoDB E2E. |
| Identical replay and changed duplicate conflict | intake unit tests plus every adapter replay/conflict test | All storage contracts and all implementation E2E targets assert `201`/`200` replay/`409` conflict. |
| Lease exclusion and expiry | adapter lifecycle tests, `TestStoreSerializesConcurrentDuplicates` | PostgreSQL/DynamoDB component contracts race two claimers and assert one winner; contract suite asserts expiry. |
| Retry schedule, success, and terminal failure | `TestDispatcherRetriesAndTerminatesFailedEvents`, store outcome/lifecycle tests | Shared real-store contracts assert retry timing, publication removal, and terminal state. |
| Authentication and partner boundary | `TestStaticAPIKeysAuthenticatesConfiguredKeys`, rejection tests, HTTP error table | Every E2E request authenticates; secret-rotation E2E rejects the old key. |
| Strict body size/shape and safe errors | `TestPostTransactionErrors`, trailing JSON and conversion failure cases | OpenAPI contract gate plus local smoke valid/replay/conflict behavior. |
| Health, readiness, metrics, method guard, resource limits | `TestOperationalEndpointsAndMethodGuard`, `TestReadinessReadyAndRequestConversionFailures`, `TestNewHTTPServerHasExplicitResourceLimits` | Compose health gating and every E2E readiness loop. |
| Graceful shutdown and cancellation | `TestServeHTTPServerGracefullyShutsDown`, `TestWorkerLoopStopsOnCancellationAndWrapsStoreFailure` | Compose restart-persistence and complete teardown assertions. |
| Kafka key/envelope/schema and safe payload | `TestPublisherWritesVersionedMessageWithStableKey`, topic/publisher config tests | Plain and TLS/SCRAM component/E2E consumers assert one versioned review event after replay. |
| Kafka trust and credential failures | writer and topic-bootstrap invalid trust/config tests | `make test-component-kafka-secure` proves correct SCRAM, wrong credentials, trusted CA, and cleanup. |
| Local/AWS secret lookup, precedence, rotation, redaction | file/AWS provider tests and command provider-selection tests | `make test-component-secrets` and `make test-e2e-secrets` exercise restart-shaped rotation and log leakage checks. |
| PostgreSQL migration race and dependency failure | migration/schema/rollback/query error tests | Compose PostgreSQL contract and separate API/worker E2E, including unavailable database response. |
| DynamoDB table race, conditional writes, malformed records, dependency failure | EnsureTable, conditional lease/write, malformed/result error tests | DynamoDB Local contract and separate API/worker E2E, including unavailable service response. |
| Infrastructure security and no plaintext secrets | deterministic infrastructure policy script and offline manifest validator | Credential-free Terraform validation/plan plus Kustomize rendering; no apply or cluster required. |
| Cleanup/isolation | script traps and command/container contracts | Aggregate component/E2E gates assert no project containers, volumes, networks, ports, generated credentials, or seeded files remain. |

Fixtures remain explicit: domain valid/invalid cases are named subtests, HTTP error tables carry their own bodies/statuses, shared store acceptance is contract-valid before mutation, and secure Kafka material is generated only in the self-cleaning fixture directory. Test helpers do not repair invalid values.
