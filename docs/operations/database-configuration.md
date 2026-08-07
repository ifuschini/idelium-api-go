# Database Configuration and Secret Loading

## Precedence

The Go API accepts Idelium-native variables and the compatible Laravel/Docker
aliases during coexistence. Values are resolved in this order:

1. `IDELIUM_DB_*` variable;
2. compatible `DB_*` variable;
3. documented non-secret default.

For the password, `IDELIUM_DB_PASSWORD_FILE` or `DB_PASSWORD_FILE` takes
precedence over the corresponding environment value. If a configured secret
file is missing, unreadable, or empty, startup fails; it does not silently use a
fallback password. New production deployments must mount a secret file.

## Validation

Startup rejects:

- missing host, database name, user, or password;
- ports outside `1..65535`;
- non-positive connection, read, write, or pool-lifetime timeouts;
- invalid pool sizes or an idle limit larger than the open limit;
- TLS modes other than `false`, `true`, or `preferred`.

The pool uses bounded connect, read, and write timeouts, parses MySQL timestamps,
and uses `utf8mb4_unicode_ci` for compatibility with the existing database.

## Safe diagnostics

Configuration and connection failures expose only a stable classification such
as unreadable secret file, deadline exceeded, network failure, MySQL numeric
error, or database unavailable. They do not include the password, DSN, secret
file path, authorization data, or raw driver error. Operational systems should
correlate the failure with infrastructure telemetry rather than enabling verbose
credential-bearing driver logs.

## Compatibility, deployment, and rollback

This configuration layer opens and checks the existing MySQL service but changes
no schema and performs no application write. The existing Laravel variables
remain supported, and Laravel retains all business-route ownership. OpenAPI,
Laravel-Go differential, and cross-tenant route tests are not applicable to this
startup-only slice; MySQL connectivity is covered by the integration suite.

Deploy by mounting the password secret, validating readiness, and keeping the
previous immutable image available. Rollback drains the Go instance and restores
that image. No database restoration or reverse migration is required.
