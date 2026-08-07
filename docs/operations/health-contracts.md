# Health and Readiness Contracts

## Liveness

`GET /health/live` proves that the API process can serve HTTP. It returns `200`
with `status`, service name, version, and source commit. It deliberately does not
query MySQL or any external dependency, so an orchestrator does not restart a
healthy process during a dependency outage.

## Readiness

`GET /health/ready` performs a MySQL ping within a two-second child deadline.
It returns:

- `200` with `status`, build identity, and `dependencies.database = "ok"` when
  the dependency is available;
- `503` with the stable `DEPENDENCY_UNAVAILABLE` error envelope otherwise.

The failure response does not expose the driver error, password, DSN, host,
tenant information, or schema details. Both endpoints and both readiness outcomes
return JSON with `Cache-Control: no-store`.

The authoritative public schemas and response headers are in
[`api/openapi.yaml`](../../api/openapi.yaml).

## Orchestrator use

- Use liveness only to decide whether the process must restart.
- Use readiness to admit or remove an instance from service.
- Allow the readiness timeout to complete within the orchestrator probe timeout.
- Do not publish either endpoint through a cache or use it as a business health
  dashboard.

## Compatibility, deployment, and rollback

These operational endpoints are Go-specific and do not replace a Laravel
business route. They read no tenant-owned resource and perform no write, so
Laravel-Go differential and cross-tenant tests are not applicable. The readiness
path is verified against pinned MariaDB by the integration suite.

Deploy the endpoint with the immutable Go image and require readiness before
traffic admission. Rollback removes the Go instance and restores the previous
image; Laravel remains the business-route fallback and no database restoration
is required.
