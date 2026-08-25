# CLI Plugin Reads

This document describes the Go-owned read paths for legacy Idelium CLI plugin
configuration endpoints.

## Contract

The Go API owns:

- `GET /ideliumcl/plugins/{idProject}`
- `GET /ideliumcl/plugin/{idPlugin}`

The gateway exposes the routes as:

- `GET /api/ideliumcl/plugins/{idProject}`
- `GET /api/ideliumcl/plugin/{idPlugin}`

The routes preserve the Laravel response shapes returned by
`IdeliumClController@getPlugins` and `IdeliumClController@getPlugin`:

- plugin-list reads return an array and use `[]` when no tenant-owned plugin is
  available for the requested project;
- single plugin reads return the raw `plugins` row fields used by the CLI;
- missing, malformed, and cross-tenant single-plugin identifiers return `404`
  with `{"message":"Invalid id"}`;
- missing, malformed, expired, or invalid legacy API keys return `401` with
  `{"message":"Invalid key"}` through the shared legacy CLI key middleware.

## Tenant boundary

The repository resolves plugin data only when customer ownership is part of the
same SQL query:

- list reads require `plugins.idProject = {idProject}` and
  `plugins.idCostumer = authenticatedCustomer.id`;
- single reads require `plugins.id = {idPlugin}` and
  `plugins.idCostumer = authenticatedCustomer.id`.

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

1. Change `GET|HEAD /api/ideliumcl/plugins/{idProject}` and
   `GET|HEAD /api/ideliumcl/plugin/{idPlugin}` from `go-owned` to
   `laravel-application`.
2. Regenerate the compatibility backlog, ownership matrix, smoke targets, and
   OpenAPI server contract.
3. Route traffic for the endpoints back to Laravel.

The implementation performs only reads and does not introduce dual writes.
