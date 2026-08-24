# Platform status read migration

Wave 3 moves the read-only platform status catalog from Laravel to the Go API
without changing the public route contract consumed by Idelium Web.

## Owned route

| Method | Public path | Go handler | Rollout status |
| --- | --- | --- | --- |
| `GET` | `/api/admin/platforms/status` | `platforms.Handler.Statuses` | `go-owned` |

The OpenAPI document exposes the gateway path as `/admin/platforms/status`; the
existing `/api` public prefix remains a gateway concern.

## Data access contract

The handler uses `CatalogRepository.ListStatuses` and preserves the legacy array
shape:

```json
[
  {
    "id": 1,
    "name": "free"
  }
]
```

The MySQL implementation performs a deterministic safe read:

```sql
SELECT id, name FROM statuses ORDER BY id ASC
```

The route performs no writes, has no side effects, and does not expose
credentials or tenant payload data.

## Tenant isolation note

Platform statuses are shared catalog metadata. Browser-session authorization is
still required, but the result set is not a tenant-owned resource. Tenant
ownership applies when statuses are attached to project-specific platform
targets or execution records.

## Contract and smoke evidence

Ownership is declared in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json).
The generated compatibility backlog, ownership matrix, and Web smoke target plan
route this safe read to `IDELIUM_WEB_SMOKE_GO_BASE_URL`.

Required verification:

```sh
make openapi-check
make smoke-targets-check
python3 -m unittest discover -s tests -p 'test_*.py'
```

`make verify` remains the complete gate when Go is installed locally.

## Rollback

Rollback is route-level and does not require a schema change:

1. Change `GET|HEAD /api/admin/platforms/status` in
   `docs/contracts/route-rollout-overrides.json` from `go-owned` back to
   `laravel-owned`, or remove the override.
2. Regenerate the compatibility backlog, ownership matrix, and smoke targets.
3. Route `/api/admin/platforms/status` back to Laravel at the gateway.
