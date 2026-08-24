# Platform type read migration

Wave 3 starts by moving the read-only platform type catalog from Laravel to the
Go API while keeping the browser-facing route contract unchanged.

## Owned route

| Method | Public path | Go handler | Rollout status |
| --- | --- | --- | --- |
| `GET` | `/api/admin/platforms/types` | `platforms.Handler.Types` | `go-owned` |

The committed OpenAPI contract keeps the gateway path as
`/admin/platforms/types`; the Docker or ingress routing layer is responsible for
preserving the existing `/api` public prefix.

## Data access contract

The handler uses `CatalogRepository.ListTypes`, which returns the legacy response
shape expected by Idelium Web:

```json
[
  {
    "id": 1,
    "name": "desktop"
  }
]
```

The MySQL implementation reads from the immutable platform catalog table with a
stable ordering:

```sql
SELECT id, name FROM types ORDER BY id ASC
```

The endpoint does not write data, does not expose credentials, and does not
serialize persistence models directly.

## Tenant isolation note

Platform types are global catalog metadata used to describe execution targets.
The route is still browser-session protected and carries tenant context through
the request path, but the returned rows are not customer-owned records. Tenant
isolation for platform execution data remains enforced at the routes that attach
platform targets to projects, test launches, and execution results.

## Contract and smoke evidence

The route ownership is declared in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json)
and then regenerated through the compatibility backlog, ownership matrix, and Web
smoke target plan. The Web smoke target now points this route at
`IDELIUM_WEB_SMOKE_GO_BASE_URL`.

Required verification:

```sh
make openapi-check
make smoke-targets-check
python3 -m unittest discover -s tests -p 'test_*.py'
```

`make verify` remains the full local gate when Go is available.

## Rollback

Rollback is a route-level ownership revert:

1. Change `GET|HEAD /api/admin/platforms/types` in
   `docs/contracts/route-rollout-overrides.json` from `go-owned` back to
   `laravel-owned`, or remove the override.
2. Regenerate the compatibility backlog, ownership matrix, and smoke targets.
3. Route `/api/admin/platforms/types` back to Laravel at the ingress layer.

No database migration is required for rollback because the Go route reads the
existing Laravel-compatible catalog table.
