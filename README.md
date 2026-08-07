# Idelium API Go

`idelium-api-go` is the contract-compatible Go replacement for the Idelium
Laravel API. It is being delivered incrementally so Idelium Web, Idelium CLI,
Idelium runners, existing databases, and Docker deployments remain compatible
throughout the migration.

The migration strategy is documented in [MIGRATION_PLAN.md](MIGRATION_PLAN.md)
and translated into executable epics in
[docs/migration/epics.md](docs/migration/epics.md).
The current migration cursor and GitHub issue mapping are maintained in
[docs/migration/progress.md](docs/migration/progress.md).
Consequential architecture choices are indexed in
[docs/adr/README.md](docs/adr/README.md), beginning with the accepted
[route-level strangler migration model](docs/adr/0001-strangler-migration-model.md).

## Current scope

The repository currently provides the production foundation for the new API:

- validated configuration with Docker secret-file support;
- structured, redacted request logging;
- correlation identifiers and a stable error envelope;
- liveness and MySQL-backed readiness endpoints;
- read-only legacy-compatible platform type and status catalog endpoints;
- bounded HTTP and database timeouts;
- graceful shutdown;
- unit, race, and container build gates;
- the initial OpenAPI contract and Laravel route inventory.

The complete, reproducible Laravel route baseline is available in
[docs/contracts/laravel-routes.md](docs/contracts/laravel-routes.md), with its
machine-readable counterpart in
[docs/contracts/laravel-routes.json](docs/contracts/laravel-routes.json).
The [route consumer map](docs/contracts/consumer-route-map.md) records which
Laravel routes are used by Idelium Web, CLI, runners, Docker operations, and
the published wiki workflows, including unresolved consumer references.
The generated
[compatibility contract backlog](docs/contracts/compatibility-backlog.md)
tracks the evidence required before each production-visible Laravel route can
move to Go.
The [sanitized golden fixture policy](docs/contracts/golden-fixture-policy.md)
defines the bounded, synthetic evidence format used by future Laravel-versus-Go
differential tests and provides a validator that prevents sensitive headers,
credentials, and real tenant identities from entering committed fixtures.
The generated
[migration ownership matrix](docs/contracts/migration-ownership-matrix.md)
assigns every production-visible route to a transaction-owning aggregate,
records its effective owner, and enforces the no-dual-write migration rule.
The [route-switch gate policy](docs/operations/route-switch-gates.md) defines
required compatibility evidence, independent approvals, progressive rollout
stages, stop thresholds, and the Laravel rollback procedure.

No Laravel business route is owned by Go yet. Route ownership will move only
after its compatibility and tenant-isolation gates pass.

The bounded startup, serving, signal, and shutdown behavior is documented in
[docs/operations/runtime-lifecycle.md](docs/operations/runtime-lifecycle.md).
Fail-closed [worker and migration process skeletons](docs/operations/process-skeletons.md)
are compiled for future owning domains but are not shipped in the API image or
allowed to run before a handler and ownership boundary are registered.
The mandatory repository assets and the automated checks that protect them are
documented in
[docs/operations/repository-foundation.md](docs/operations/repository-foundation.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health/live` | Process liveness and build identity. |
| `GET` | `/health/ready` | Readiness including a bounded MySQL ping. |
| `GET` | `/admin/platforms/types` | Legacy-compatible read-only platform type catalog. |
| `GET` | `/admin/platforms/status` | Legacy-compatible read-only platform status catalog. |

Health and catalog error responses never include database connection strings or
credentials.
Health probe semantics, caching rules, and orchestrator guidance are documented
in [docs/operations/health-contracts.md](docs/operations/health-contracts.md).
The shared correlation, redacted logging, panic recovery, and security-header
contract is documented in
[docs/operations/http-safety-observability.md](docs/operations/http-safety-observability.md).

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `IDELIUM_HTTP_ADDRESS` | `:8080` | HTTP listen address. |
| `IDELIUM_HTTP_READ_HEADER_TIMEOUT` | `5s` | Header read timeout. |
| `IDELIUM_HTTP_READ_TIMEOUT` | `15s` | Request read timeout. |
| `IDELIUM_HTTP_WRITE_TIMEOUT` | `30s` | Response write timeout. |
| `IDELIUM_HTTP_IDLE_TIMEOUT` | `60s` | Keep-alive idle timeout. |
| `IDELIUM_HTTP_SHUTDOWN_TIMEOUT` | `15s` | Graceful shutdown timeout. |
| `IDELIUM_DB_HOST` / `DB_HOST` | `127.0.0.1` | MySQL host. |
| `IDELIUM_DB_PORT` / `DB_PORT` | `3306` | MySQL port. |
| `IDELIUM_DB_NAME` / `DB_DATABASE` | `ideliumdb` | MySQL database. |
| `IDELIUM_DB_USER` / `DB_USERNAME` | `idelium` | MySQL user. |
| `IDELIUM_DB_PASSWORD_FILE` / `DB_PASSWORD_FILE` | — | Preferred password secret file. |
| `IDELIUM_DB_PASSWORD` / `DB_PASSWORD` | — | Password fallback for local development. |
| `IDELIUM_DB_CONNECT_TIMEOUT` | `5s` | Connection timeout. |
| `IDELIUM_DB_READ_TIMEOUT` | `10s` | Read timeout. |
| `IDELIUM_DB_WRITE_TIMEOUT` | `10s` | Write timeout. |
| `IDELIUM_DB_MAX_OPEN_CONNECTIONS` | `25` | Maximum open connections. |
| `IDELIUM_DB_MAX_IDLE_CONNECTIONS` | `10` | Maximum idle connections. |
| `IDELIUM_DB_CONNECTION_MAX_LIFETIME` | `5m` | Maximum connection lifetime. |

Production deployments should provide the database password through a mounted
secret file. Configuration errors stop startup with safe diagnostics.
Detailed precedence, validation, redaction, and rollback behavior is documented
in [docs/operations/database-configuration.md](docs/operations/database-configuration.md).
The isolated MariaDB harness and its coverage boundaries are documented in
[docs/operations/mysql-integration-testing.md](docs/operations/mysql-integration-testing.md).

## Local verification

Go is expected to be installed locally, or commands can be run through the
pinned builder image in the Dockerfile.

```sh
make verify
make integration-test
docker build -t idelium/api-go:local .
```

Run the service with a reachable MySQL instance:

```sh
IDELIUM_DB_PASSWORD_FILE=/run/secrets/db_password \
go run ./cmd/api
```

## Architecture

The service is a modular monolith. HTTP handlers depend on application services,
application services depend on tenant-scoped repository interfaces, and MySQL
implementations remain inside persistence packages. Generated API and query code
will be committed for reproducible reviews.

## License

Licensed under the Apache License 2.0. See [LICENSE](LICENSE).
