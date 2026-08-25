# Platform browser version read migration

Wave 3 moves the read-only browser version endpoint from Laravel to the Go API
while browser-version mutations remain on Laravel for Wave 6.

## Owned route

| Method | Public path | Go handler | Rollout status |
| --- | --- | --- | --- |
| `GET` | `/api/admin/platforms/browserversions/{idBrowser}` | `platforms.Handler.BrowserVersions` | `go-owned` |

`POST` and `PUT` on `/api/admin/platforms/browserversions` stay
`laravel-owned` because they mutate the catalog and require the Wave 6 mutation
contract.

## Compatibility note

The public Laravel route is intentionally spelled `browserversions`. The Go API
keeps that path unchanged for compatibility with Idelium Web and any existing
browser-session clients.

## Data access contract

The Go route preserves Laravel's `EnterpriseGridResponse` shape for browser
versions scoped to a browser row:

- `idBrowser` is validated as a positive integer at the HTTP boundary;
- without `page` or `pageSize`, the response is the legacy array of
  browser-version rows;
- with `page` or `pageSize`, the response is `{ "data": [...], "meta": {...} }`;
- each row exposes `id`, `version`, `idBrowser`, `created_at`, and `updated_at`;
- `sort` is allowlisted to `id`, `version`, `created_at`, and `updated_at`;
- the default sort is `version asc`, matching the Laravel controller;
- `direction` is bounded to `asc` or `desc`;
- `pageSize` is bounded to the Laravel-compatible `1..100` range;
- `search` is trimmed and limited to 200 characters before applying `LIKE`;
- `filter[id]` accepts positive numeric identifiers only.

The repository reads the existing Laravel-compatible table:

```sql
SELECT id, version, idBrowser, created_at, updated_at
FROM version_browsers
WHERE idBrowser = ?
ORDER BY <allowlisted-column> <allowlisted-direction>
```

Only allowlisted column names and directions are interpolated into SQL. Browser
scope, filter values, search text, limits, and offsets are bound parameters.

## Tenant isolation note

Browser versions are global platform catalog metadata under a parent browser,
not tenant-owned records. Browser-session authorization and tenant-context
metadata are preserved at the route contract, while tenant ownership is enforced
when a browser version is attached to project-scoped platform targets or
execution results.

## Contract and smoke evidence

Ownership is declared in
[`docs/contracts/route-rollout-overrides.json`](../contracts/route-rollout-overrides.json).
The generated compatibility backlog, ownership matrix, and Web smoke target plan
route the safe `GET` to `IDELIUM_WEB_SMOKE_GO_BASE_URL`, while browser-version
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

1. Change `GET|HEAD /api/admin/platforms/browserversions/{idBrowser}` in
   `docs/contracts/route-rollout-overrides.json` from `go-owned` back to
   `laravel-owned`, or remove the override.
2. Regenerate the compatibility backlog, ownership matrix, and smoke targets.
3. Route `GET /api/admin/platforms/browserversions/{idBrowser}` back to Laravel
   at the gateway.

No database migration is required because the Go route reads the existing
Laravel-compatible `version_browsers` table.
