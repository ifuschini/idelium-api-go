# MySQL Integration Testing

The Wave 1 integration harness validates the Go database boundary against the
pinned MariaDB version used for Laravel coexistence. Run it with:

```sh
make integration-test
```

The command creates an isolated Compose network and ephemeral `idelium_test`
database, waits for the database health check, runs the MySQL package tests in
the pinned Go builder, and removes containers, volumes, and the network on exit.
The source tree is mounted read-only in the test container.

## Covered contracts

The suite verifies:

- connection and bounded readiness against a real MySQL-compatible server;
- the HTTP readiness success contract backed by that connection;
- use of the dedicated `idelium_test` schema rather than a developer or
  production database;
- safe authentication failures that do not expose usernames or passwords;
- legacy-compatible platform catalog reads used by the existing read-only
  migration slice.

The committed credentials are synthetic, local integration-test values. They
must never be reused in a deployment. Production configuration continues to
prefer mounted secret files.

## Compatibility, tenant isolation, and migrations

This infrastructure does not change the legacy schema, migrate a business
route, or access tenant-owned tables. A Laravel-Go differential test, schema
migration test, and negative cross-tenant test therefore have no applicable
behavior to compare in this ticket. Each future database-backed resource must
add its own migration, differential, and negative cross-tenant coverage before
route ownership can move.

Rollback removes the integration harness only; it has no runtime or production
data effect.
