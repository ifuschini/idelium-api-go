# Parallel run worker claim cutover

Issue #151 moves browser-session and customer API-key worker claims to Go. The
claim transaction locks the tenant-and-project-scoped schedule row before it
checks the state and concurrency limit, so competing requests cannot occupy the
same slot. Retrying the same worker identifier renews its default 120-second
lease without increasing the worker count.

## Compatibility and security

- `IDELIUM_RUN_TOKEN_REQUIRED_FOR_CLAIM` remains enabled unless it is explicitly
  set to `false`, matching Laravel's default. Existing Laravel-issued `idrt_`
  tokens remain compatible and are consumed once under a row lock before the
  claim transaction.
- Token validation scopes the token to customer, project, schedule, and worker.
  Success and rejection audits contain only redacted token markers; raw tokens,
  hashes, request headers, and capabilities are never logged.
- The schedule lookup includes customer, project, and schedule ownership in the
  locked SQL query. Missing and cross-tenant schedules return the same not-found
  response.
- Registered agents must be approved and not unhealthy. A configured
  certificate SHA-256 identity proof is compared without case sensitivity and
  fails closed when missing or different.
- Terminal schedules return 422, cancelling schedules and exhausted concurrency
  return 409, and invalid or consumed tokens use the Laravel validation envelope.

The Laravel `ParallelRunScheduleApiTest` and the claim-focused `RunTokenTest`
cases are the differential references. The Go MySQL integration test starts two
claims simultaneously against one available slot and requires exactly one
success and one concurrency rejection.

## Deployment and rollback

Deploy the pinned Go image after MySQL integration, Laravel differential, and
race tests pass. Route only the two claim operations to Go; heartbeat, worker
updates, cancellation, results, token administration, and runner-only routes
remain on Laravel until their Wave 8 tickets complete. Laravel may continue to
issue compatible run tokens from the shared database during this slice.

Rollback routes the two claim operations to Laravel. No schema rollback, data
conversion, reverse replay, or dual write is required. Claims and token
consumption committed before rollback remain valid shared state and must not be
replayed with a fresh token automatically.
