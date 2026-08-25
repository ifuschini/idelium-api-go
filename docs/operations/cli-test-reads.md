# CLI Test Reads

This document describes the Go-owned read path for the legacy Idelium CLI test
configuration endpoint.

## Contract

The Go API owns:

- `GET /ideliumcl/test/{idTest}`

The gateway exposes the route as:

- `GET /api/ideliumcl/test/{idTest}`

The route preserves the Laravel response shape returned by
`IdeliumClController@getTest`:

- successful reads return the raw `tests` row fields used by the CLI;
- missing, malformed, and cross-tenant identifiers return `404` with
  `{"message":"Invalid id"}`;
- missing, malformed, expired, or invalid legacy API keys return `401` with
  `{"message":"Invalid key"}` through the shared legacy CLI key middleware.

## Tenant boundary

The repository resolves a test only when both conditions match in the same SQL
query:

- `tests.id = {idTest}`;
- `tests.idCostumer = authenticatedCustomer.id`.

Foreign-tenant rows intentionally look identical to missing rows. This preserves
the legacy CLI behavior and avoids tenant enumeration.

## Diagnostics and redaction

Server logs include only safe request metadata such as the correlation id, route
path, and failure class. Database errors are wrapped through the shared safe
database diagnostic helper, and the legacy `Idelium-Key` header is never logged
or persisted in golden fixtures.

## Deployment and rollback

Route ownership is recorded in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json).

Rollback is route-level and does not require a schema migration:

1. Change `GET|HEAD /api/ideliumcl/test/{idTest}` from `go-owned` to
   `laravel-application`.
2. Regenerate the compatibility backlog, ownership matrix, smoke targets, and
   OpenAPI server contract.
3. Route traffic for the endpoint back to Laravel.

The implementation performs only reads and does not introduce dual writes.
