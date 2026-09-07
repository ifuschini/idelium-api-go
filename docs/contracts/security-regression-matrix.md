# Parallel-run Security Regression Matrix

This matrix is the evidence index for issue [#192](https://github.com/ifuschini/idelium-api-go/issues/192).
The same synthetic tenant and token cases are maintained in both runtimes;
the test sources below are executed by their respective Docker/CI jobs.

| Scenario | Laravel evidence | Go evidence |
| --- | --- | --- |
| Unauthenticated browser session | `idelium-api/tests/Feature/BrowserSessionAuthenticationTest.php` (`test_logout_requires_authentication_and_invalidates_the_session`) | `idelium-api-go/internal/browserauth/handler_test.go` (`TestLogoutRequiresSessionAndDeletesOpaqueIdentifier`) |
| Missing/invalid API key | `idelium-api/tests/Feature/IdeliumCliTenantIsolationTest.php` (`test_cli_routes_reject_a_missing_or_invalid_api_key`) | `idelium-api-go/internal/auth/legacy_key_test.go` (`TestLegacyKeyAuthenticatorRejectsMissingKeyWithLaravelCompatibleBody`, `TestLegacyKeyAuthenticatorRejectsInvalidKeyWithRedactedDiagnostics`) |
| Cross-tenant reads and mutations | `idelium-api/tests/Feature/ParallelRunScheduleApiTest.php` and `IdeliumCliTenantIsolationTest.php` | `idelium-api-go/internal/persistence/mysql/database_integration_test.go` (cross-tenant schedule, claim, heartbeat, cancellation, and CLI assertions) |
| Run-token expiry, reuse, wrong agent, and revocation | `idelium-api/tests/Feature/RunTokenTest.php` | `idelium-api-go/internal/browserauth/handler_test.go` and `internal/persistence/mysql/database_integration_test.go` |
| Worker-token missing, expired, foreign, or wrong worker | `idelium-api/tests/Feature/RunTokenTest.php` (`test_token_only_runner_heartbeat_rejects_invalid_worker_token`) | `idelium-api-go/internal/browserauth/handler_test.go` and `internal/persistence/mysql/worker_token_test.go` |
| Hashing and secret redaction | `idelium-api/tests/Feature/RunTokenTest.php` (`test_run_token_is_revealed_once_and_stored_as_hash`) | `idelium-api-go/internal/persistence/mysql/database_integration_test.go` and `internal/auditlog/redact.go` |

## Required assertions

- Owner-tenant operations succeed and foreign-tenant operations are denied
  without revealing whether a resource exists.
- Missing, malformed, expired, consumed, revoked, and incorrectly bound tokens
  return the documented 401/404 shape.
- Plaintext tokens, API keys, cookies, authorization headers, and tenant
  identifiers are absent from responses, logs, fixtures, and audit records.
- MySQL checks prove bcrypt hashes, expiry, single-use consumption, and
  tenant/project/run/worker binding.

## Reproducible verification

Go unit and MySQL integration tests run in the pinned Go/MariaDB Compose profile
(`smoke/docker-compose.yml`). Laravel feature tests run in the pinned Composer
profile of `idelium-api`. The CI jobs must publish only redacted output; this
repository's fixture and log scanners fail if credential fields are introduced.
