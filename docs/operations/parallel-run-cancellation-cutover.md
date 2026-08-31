# Parallel run cancellation cutover

Issue #153 moves browser-session and customer API-key cancellation operations
to Go. Cancellation locks the tenant-and-project-scoped schedule row and commits
worker transitions, counters, result summary, timestamps, aggregate result, and
final schedule status atomically.

## Compatibility and convergence

- Missing and cross-tenant schedules return the same not-found response. A run
  that is already terminal returns Laravel-compatible 422 behavior, including
  repeated cancellation requests.
- Every running worker becomes `cancelled` with a single request timestamp.
  Workers that already completed, failed, or became lost retain their state and
  result.
- Laravel's deterministic precedence is preserved: failure wins, then lost;
  mixed completed and cancelled workers converge to `failed` with cancelled
  aggregate status, while an all-cancelled or empty schedule is `cancelled`.
- `cancelledAt`, `completedAt`, worker counters, ordered result summary, and the
  aggregate status are written under the same row lock. No partial cancellation
  is externally visible.
- Diagnostics contain correlation context but never include worker results,
  capabilities, tokens, headers, or other payload values.

The Laravel `ParallelRunScheduleApiTest` is the differential reference. Go unit
tests cover success, terminal retry, invalid path identifiers, and safe errors;
MySQL integration covers empty and mixed-worker convergence plus negative
cross-tenant access.

## Deployment and rollback

Deploy the pinned image after Go verification, MySQL integration, and the
Laravel reference suite pass. Route only the browser and API-key cancellation
operations to Go; worker updates, results, token administration, and runner-only
routes remain Laravel-owned until their separate Wave 8 tickets complete.

Rollback routes the two cancellation operations to Laravel. Both runtimes use
the same schedule row and worker-state JSON, so no schema rollback, data
conversion, reverse replay, or dual write is required. Do not retry a cancellation
automatically after an ambiguous client timeout; read the schedule first.
