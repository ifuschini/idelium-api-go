# OpenAPI Compatibility Contracts

Wave 2 requires every production-visible Laravel route to appear in the Go
OpenAPI document before traffic ownership can move. The contract is intentionally
split into two levels:

- native Go routes, which keep detailed request and response schemas;
- Laravel compatibility routes, which preserve externally visible route,
  authentication, tenant, consumer, and ownership metadata until the route is
  migrated.

The compatibility contracts do not transfer runtime ownership to Go and do not
claim payload parity. They make the migration surface explicit so later tickets
can attach sanitized fixtures, Laravel-Go differential checks, tenant isolation
tests, smoke tests, and rollout evidence to every route.

## Compatibility Metadata

Generated compatibility operations include:

- `x-idelium-laravel-route`: the original Laravel route including the `/api`
  prefix when present;
- `x-idelium-controller`: the Laravel controller or closure that currently owns
  the behavior;
- `x-idelium-current-owner`: the current runtime owner;
- `x-idelium-trust-path`: the migration trust path;
- `x-idelium-authentication-mode`: the expected authentication model;
- `x-idelium-tenant-context`: whether tenant ownership must be enforced;
- `x-idelium-consumers`: known external consumers from the consumer route map.

## Regeneration

Regenerate the contracts after updating the Laravel inventory or consumer map:

```sh
python3 scripts/sync_openapi_legacy_contracts.py \
  --inventory docs/contracts/laravel-routes.json \
  --consumer-map docs/contracts/consumer-route-map.json \
  --openapi api/openapi.yaml
```

Then refresh the compatibility backlog:

```sh
python3 scripts/build_compatibility_backlog.py \
  --inventory docs/contracts/laravel-routes.json \
  --consumer-map docs/contracts/consumer-route-map.json \
  --openapi api/openapi.yaml \
  --output-json docs/contracts/compatibility-backlog.json \
  --output-markdown docs/contracts/compatibility-backlog.md
```

`make verify` runs the synchronization check and fails if `api/openapi.yaml`
does not match the committed Laravel route inventory.

## Deployment and Rollback

This change is documentation and contract metadata only. It does not add route
handlers, change database schema, or change runtime traffic routing. Rollback is
a Git revert of the OpenAPI, generated backlog, script, and tests.
