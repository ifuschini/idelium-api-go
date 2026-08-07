# Worker and Migration Process Skeletons

## Current state

The repository builds three process entry points:

- `cmd/api`, the production HTTP process;
- `cmd/worker`, reserved for background jobs that move with their owning domain;
- `cmd/migrate`, reserved for reviewed Go-owned schema migrations.

The worker and migration binaries are intentionally fail-closed. They print safe
build identity with `-version`, but refuse normal execution until an owning domain
and handler are registered together. This prevents an empty worker from appearing
healthy and prevents a no-op migration command from reporting false success.

The current API container continues to include only `cmd/api`. Docker and release
artifacts must not publish or schedule either skeleton as an operational process.

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

The skeletons expose no HTTP contract, connect to no database, consume no queue,
and perform no writes. OpenAPI, differential, MySQL, and cross-tenant execution
are therefore not applicable until activation. The deployment change is limited
to compiling the entry points in verification; the runtime image remains
unchanged. Rollback is a Git revert, with Laravel retaining all work and migration
ownership.
