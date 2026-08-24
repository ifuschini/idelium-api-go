# Platform location read migration

Wave 3 moves the read-only location catalog endpoint from Laravel to the Go API
while leaving location mutations on Laravel for Wave 6.

## Owned route

| Method | Public path | Go handler | Rollout status |
| --- | --- | --- | --- |
| `GET` | `/api/admin/platforms/locations` | `platforms.Handler.Locations` | `go-owned` |

`POST` and `PUT` on the same public path remain `laravel-owned` because they are
catalog mutations and are intentionally outside this safe-read migration slice.

## Data access contract

The Go route preserves Laravel's `EnterpriseGridResponse` behavior:

- without `page` or `pageSize`, the response is the legacy array of location
  rows;
- with `page` or `pageSize`, the response is `{ "data": [...], "meta": {...} }`;
- `sort` is allowlisted to `id`, `name`, `created_at`, and `updated_at`;
- `direction` is bounded to `asc` or `desc`;
- `pageSize` is bounded to the Laravel-compatible `1..100` range;
- `search` is trimmed and limited to 200 characters before applying `LIKE`;
- `filter[id]` accepts positive numeric identifiers only.

The repository query remains deterministic and side-effect free:

```sql
SELECT id, name, created_at, updated_at
FROM locations
WHERE ...
ORDER BY <allowlisted-column> <allowlisted-direction>
```

The implementation never interpolates untrusted sort or direction values into
SQL. Filter values, search text, limits, and offsets are bound parameters.

## Tenant isolation note

Locations are global platform catalog metadata. The route remains
browser-session protected and carries the existing tenant context metadata, but
the rows are not customer-owned resources. Tenant ownership is enforced when a
platform location is attached to a project-specific target or execution result.

## Contract and smoke evidence

Ownership is declared in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json).
The generated compatibility backlog, ownership matrix, and Web smoke target plan
route the safe `GET` to `IDELIUM_WEB_SMOKE_GO_BASE_URL`, while `POST` and `PUT`
continue to target Laravel.

Required verification:

```sh
make openapi-check
make smoke-targets-check
python3 -m unittest discover -s tests -p 'test_*.py'
```

`make verify` remains the complete local gate when Go is installed.

## Rollback

Rollback is route-level:

1. Change `GET|HEAD /api/admin/platforms/locations` in
   `docs/contracts/route-rollout-overrides.json` from `go-owned` back to
   `laravel-owned`, or remove the override.
2. Regenerate the compatibility backlog, ownership matrix, and smoke targets.
3. Route `GET /api/admin/platforms/locations` back to Laravel at the gateway.

No database migration is required because the Go route reads the existing
Laravel-compatible `locations` table.
