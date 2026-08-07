# API Runtime Lifecycle

## Startup

The API loads and validates all configuration before opening a listener. It then
opens the MySQL pool and performs a bounded startup check. Invalid configuration,
an unavailable database, or a listener error stops startup with a nonzero exit
code and a diagnostic that excludes credentials and connection strings.

The release build injects `version` and `commit` through Go linker flags. These
safe fields are written to the structured startup log and returned by the
versioned liveness contract. Container builds also record the source revision as
an OCI label.

## Serving

The HTTP server applies independent positive bounds for:

- request-header reads;
- complete request reads;
- response writes;
- idle keep-alive connections;
- graceful shutdown.

The server accepts `SIGINT` and `SIGTERM` through a cancellable lifecycle
context. The listener and server lifecycle are isolated in `internal/server` so
startup failures, configured bounds, and cancellation can be tested without a
production database.

## Shutdown

On cancellation, the server stops accepting new connections and allows active
requests to finish within `IDELIUM_HTTP_SHUTDOWN_TIMEOUT`. If that bound expires,
remaining connections are closed and the process exits with an error. The MySQL
pool closes after HTTP serving has stopped.

Orchestrators must remove the instance from service before sending the shutdown
signal and provide a termination grace period longer than the configured
shutdown timeout.

## Compatibility, deployment, and rollback

The lifecycle exposes only the already documented `/health/live` and
`/health/ready` contracts. It does not add a business route, modify the database,
or move traffic ownership from Laravel. Laravel-Go differential testing and
cross-tenant database coverage are not applicable to this process-only slice.

Deploy the immutable image by digest, wait for readiness, and then admit traffic.
Rollback drains the Go instance and restores the previous immutable Go image;
Laravel remains the route fallback throughout Wave 1. No data restoration or
write replay is required.
