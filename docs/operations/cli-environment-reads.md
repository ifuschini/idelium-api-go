# CLI Environment Reads

This document describes the Go-owned read paths for legacy Idelium CLI
environment configuration endpoints.

## Contract

The Go API owns:

- `GET /ideliumcl/environments/{idProject}`
- `GET /ideliumcl/environment/{idEnvironment}`

The gateway exposes the routes as:

- `GET /api/ideliumcl/environments/{idProject}`
- `GET /api/ideliumcl/environment/{idEnvironment}`

The routes preserve the Laravel response shapes returned by
`IdeliumClController@getEnvironments` and `IdeliumClController@getEnvironment`:

- environment-list reads return an array and use `[]` when no tenant-owned
  environment is available for the requested project;
- single environment reads return the raw `environments` row fields used by the
  CLI, including the legacy `idCostumer` spelling;
- missing, malformed, and cross-tenant single-environment identifiers return
  `404` with `{"message":"Invalid id"}`;
- missing, malformed, expired, or invalid legacy API keys return `401` with
  `{"message":"Invalid key"}` through the shared legacy CLI key middleware.

## Tenant boundary

The repository resolves environment data only when customer ownership is part of
the same SQL query:

- list reads require `environments.idProject = {idProject}` and
  `environments.idCostumer = authenticatedCustomer.id`;
- single reads require `environments.id = {idEnvironment}` and
  `environments.idCostumer = authenticatedCustomer.id`.

Foreign-tenant rows intentionally look identical to empty or missing resources.
This preserves the legacy CLI behavior and avoids tenant enumeration.

## Diagnostics and redaction

Server logs include only safe request metadata such as the correlation id, route
path, and failure class. Database errors are wrapped through the shared safe
database diagnostic helper, and the legacy `Idelium-Key` header is never logged
or persisted in golden fixtures.

## Deployment and rollback

Route ownership is recorded in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json).

Rollback is route-level and does not require a schema migration:

1. Change `GET|HEAD /api/ideliumcl/environments/{idProject}` and
   `GET|HEAD /api/ideliumcl/environment/{idEnvironment}` from `go-owned` to
   `laravel-application`.
2. Regenerate the compatibility backlog, ownership matrix, smoke targets, and
   OpenAPI server contract.
3. Route traffic for the endpoints back to Laravel.

The implementation performs only reads and does not introduce dual writes.
