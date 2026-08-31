# Laravel queue drain and integration worker cutover

Issue #149 provides a fail-closed queue-drain verifier and activates the Go
integration-delivery worker. This is an operational contract, not an HTTP API,
so OpenAPI and request-level Laravel-Go differential testing are not applicable.

## Drain procedure

1. Block Laravel routes and schedulers that can enqueue integration delivery,
   result export, or artifact purge work.
2. Keep the existing Laravel workers running until their normal queue is empty.
3. Stop every Laravel queue worker. Do not start the Go worker yet.
4. Resolve or explicitly archive failed Laravel jobs and allow domain rows in
   `pending`, retryable `failed`, or `queued` states to finish.
5. Run the aggregate verifier against the shared database:

   ```sh
   /idelium-migrate --verify-laravel-queue-drain \
     --laravel-queue-driver=sync \
     --confirm-laravel-workers-stopped
   ```

   Use `database` instead of `sync` when Laravel used its database queue. A
   ready result requires zero rows in the database queue, `failed_jobs`, pending
   or retryable integration deliveries, and queued result exports. Exit status
   2 means the drain is still blocked; status 1 means configuration or database
   inspection failed safely.
6. Capture the JSON result as release evidence. It contains aggregate counts
   only and never reads tenant identifiers, serialized payloads, exceptions, or
   secrets.
7. Start exactly one `/idelium-worker` process with
   `IDELIUM_INTEGRATION_WORKER_ENABLED=true`, the shared database configuration,
   and the Laravel-compatible `APP_KEY` secret. Then move integration mutation
   routes to Go.

The worker also acquires a global MySQL advisory lease. A second worker exits
before consuming work, providing a database-enforced single-consumer gate even
if deployment replica configuration is incorrect. Go-created integration rows
carry schema version `2026-07-28.v1` and the worker sends that version in the
signed adapter contract.

## Verification scope

Unit tests cover ready, blocked, invalid-driver, missing-secret, polling, and
safe-error behavior. MySQL integration tests cover global aggregate counts,
missing work exclusion, due-delivery selection, and exclusive worker leasing.
The existing Laravel integration tests remain the reference for delivery states
and adapter side effects; replaying a queue drain as a request differential is
not meaningful because it consumes immutable operational state.

## Rollback

First return integration mutation routes to Laravel, then stop the Go worker and
confirm its advisory lease is released. Resume the Laravel workers only after Go
has stopped. Existing pending rows remain in the shared compatible tables and do
not require reverse replay or a database restore. Never run both consumers at
the same time.
