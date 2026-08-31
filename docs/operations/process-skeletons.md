# Worker and Migration Process Skeletons

## Current state

The repository builds three process entry points:

- `cmd/api`, the production HTTP process;
- `cmd/worker`, reserved for background jobs that move with their owning domain;
- `cmd/migrate`, reserved for reviewed Go-owned schema migrations.

The migration binary remains fail-closed outside its explicit verification and
bridge modes. The worker is now registered to the integration-delivery domain,
but remains disabled unless `IDELIUM_INTEGRATION_WORKER_ENABLED=true` is set
after the Laravel queue drain. This prevents an empty or premature worker from
appearing healthy.

The pinned image contains `/idelium-api-go`, `/idelium-worker`, and the explicit
`/idelium-migrate` verification/bridge command; the API remains the default
entrypoint. Deployments must schedule the worker as a separate,
single-replica process only after the drain verifier reports ready.

## Activation gate

Activating a skeleton requires a dedicated migration ticket that:

1. names the transaction-owning domain in the ownership matrix;
2. registers a cancellable handler and its versioned work or migration format;
3. validates configuration before accepting work;
4. proves tenant ownership in the same query or transaction where applicable;
5. adds unit, MySQL integration, retry, failure, and cross-tenant tests;
6. documents observability, deployment, drain, and rollback behavior;
7. adds the pinned binary to an explicit image or one-shot deployment artifact.

Background jobs move with their owning domain. Laravel-created jobs must drain
before Go becomes the queue owner, and application-level dual consumption is
prohibited. Go migrations remain backward compatible while Laravel is a fallback.

## Diagnostics and security

Lifecycle diagnostics contain only process name, owning-domain name, version,
and source commit. They never contain database credentials, authorization
headers, cookies, session identifiers, tenant identifiers, or payloads. A
missing domain or handler exits nonzero with a stable, non-sensitive error.

## Compatibility, deployment, and rollback

The queue-drain and worker activation contract is documented in
`docs/operations/laravel-queue-drain.md`. It is not HTTP-visible, so OpenAPI is
not applicable. Rollback stops the Go worker before Laravel queue processing is
resumed; the shared compatible delivery rows require no reverse replay.
