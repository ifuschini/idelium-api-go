# Platform operating-system version read migration

Wave 3 moves the read-only operating-system version endpoint from Laravel to the
Go API while OS-version mutations remain on Laravel for Wave 6.

## Owned route

| Method | Public path | Go handler | Rollout status |
| --- | --- | --- | --- |
| `GET` | `/api/admin/platforms/osversion/{idOs}` | `platforms.Handler.OperatingSystemVersions` | `go-owned` |

`POST` and `PUT` on `/api/admin/platforms/osversion` stay `laravel-owned`
because they mutate the catalog and require the Wave 6 mutation contract.

## Data access contract

The Go route preserves Laravel's `EnterpriseGridResponse` shape for OS versions
scoped to an operating-system row:

- `idOs` is validated as a positive integer at the HTTP boundary;
- without `page` or `pageSize`, the response is the legacy array of OS-version
  rows;
- with `page` or `pageSize`, the response is `{ "data": [...], "meta": {...} }`;
- each row exposes `id`, `version`, `idOs`, `created_at`, and `updated_at`;
- `sort` is allowlisted to `id`, `version`, `created_at`, and `updated_at`;
- the default sort is `version asc`, matching the Laravel controller;
- `direction` is bounded to `asc` or `desc`;
- `pageSize` is bounded to the Laravel-compatible `1..100` range;
- `search` is trimmed and limited to 200 characters before applying `LIKE`;
- `filter[id]` accepts positive numeric identifiers only.

The repository reads the existing Laravel-compatible table:

```sql
SELECT id, version, idOs, created_at, updated_at
FROM version_os
WHERE idOs = ?
ORDER BY <allowlisted-column> <allowlisted-direction>
```

Only allowlisted column names and directions are interpolated into SQL. Operating
system scope, filter values, search text, limits, and offsets are bound
parameters.

## Tenant isolation note

OS versions are global platform catalog metadata under a parent operating
system, not tenant-owned records. Browser-session authorization and
tenant-context metadata are preserved at the route contract, while tenant
ownership is enforced when an OS version is attached to project-scoped platform
targets or execution results.

## Contract and smoke evidence

Ownership is declared in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json).
The generated compatibility backlog, ownership matrix, and Web smoke target plan
route the safe `GET` to `IDELIUM_WEB_SMOKE_GO_BASE_URL`, while OS-version
mutations continue to target Laravel.

Required verification:

```sh
make openapi-check
make smoke-targets-check
python3 -m unittest discover -s tests -p 'test_*.py'
```

`make verify` remains the complete local gate when Go is installed.

## Rollback

Rollback is route-level:

1. Change `GET|HEAD /api/admin/platforms/osversion/{idOs}` in
   `docs/contracts/route-rollout-overrides.json` from `go-owned` back to
   `laravel-owned`, or remove the override.
2. Regenerate the compatibility backlog, ownership matrix, and smoke targets.
3. Route `GET /api/admin/platforms/osversion/{idOs}` back to Laravel at the
   gateway.

No database migration is required because the Go route reads the existing
Laravel-compatible `version_os` table.
