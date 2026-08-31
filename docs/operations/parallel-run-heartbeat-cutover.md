# Parallel run worker heartbeat cutover

Issue #152 moves browser-session and customer API-key heartbeat operations to
Go. Heartbeats lock the tenant-and-project-scoped schedule row, evaluate lease
expiry against one UTC timestamp, and commit worker state, counters, result
summary, and terminal convergence atomically.

## Compatibility and isolation

- The default lease is 120 seconds. Explicit leases are limited to 15 through
  3600 seconds, matching Laravel validation.
- A lease expires only when its timestamp is strictly before the request time.
  A heartbeat at the exact expiry boundary may renew it, matching Carbon's
  `greaterThan` behavior.
- Expired running workers become `lost` with `lostAt` and `updatedAt` timestamps.
  When no active workers remain, status, aggregate status, counters, summary,
  and completion time converge in the same transaction.
- Missing and cross-tenant schedules are indistinguishable. Unknown workers
  return 404; terminal schedules return 422; already inactive leases return the
  updated schedule with a 409 and `workerStatus`.
- Heartbeat requests and diagnostics never log worker state, capabilities,
  tokens, headers, or other payload values.

The Laravel `ParallelRunScheduleApiTest` is the differential reference. Go unit
tests cover validation and safe diagnostics; MySQL integration covers renewal,
strict expiry boundaries, lost-worker convergence, unknown workers, and negative
cross-tenant access.

## Deployment and rollback

Deploy the pinned image only after Go verification, MySQL integration, and the
Laravel reference suite pass. Route the two heartbeat operations to Go while
worker updates, cancellation, results, and runner-only token routes remain on
Laravel until their Wave 8 tickets complete.

Rollback routes the two heartbeat operations back to Laravel. Both runtimes use
the same JSON worker-state representation and timestamps, so no schema rollback,
state conversion, reverse replay, or dual write is required.
