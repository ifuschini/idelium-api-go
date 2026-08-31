# Parallel run schedule and matrix cutover

Issue #150 moves schedule list, create, matrix-create, and detail reads to Go for
both browser sessions and legacy customer API keys. Worker claim, heartbeat,
cancellation, result, token, and agent routes remain Laravel-owned until their
separate Wave 8 tickets complete.

## Compatibility and isolation

- Every query and transaction scopes the active customer and project. Schedule
  creation locks both the owned project and owned test cycle before writing.
- The existing unique customer/project/idempotency key remains the source of
  retry safety. Matrix keys use the Laravel-compatible sorted JSON combination
  digest and all generated rows commit in one transaction.
- Concurrency is limited to 32. Each axis is limited to 16 distinct scalar
  values and the Cartesian expansion is limited to 64 schedules.
- Metadata is normalized to the immutable run envelope and sensitive keys are
  removed recursively. Each new schedule captures the current test-cycle, test,
  step, and environment asset-version references.
- List reads preserve the Laravel order and 50-row bound before applying build,
  commit, branch, repository, initiator, and pipeline filters.

The same repository implementation serves both authentication paths, so a CLI
key and browser session see the same response contract for the same tenant. The
Laravel `ParallelRunScheduleApiTest` is the differential reference; Go handler
and MySQL tests cover the migrated subset without invoking later worker-state
routes.

## Deployment and rollback

Deploy the pinned image, run MySQL integration and Laravel reference tests, then
switch only these eight operations to Go. Do not switch claim or lifecycle
routes as part of this ticket. Monitor not-found, validation, matrix-bound, and
transaction errors by correlation identifier without logging metadata payloads.

Rollback routes the eight operations back to Laravel. No schema rollback,
database restore, reverse replay, or dual write is required because both
runtimes use the existing idempotent `parallel_run_schedules` table.
