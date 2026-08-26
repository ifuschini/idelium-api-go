# CLI Performed Cycle Writes

## Scope

The Go API now owns the Laravel-compatible CLI performed-cycle write routes:

- `POST /api/ideliumcl/testcycle`
- `PUT /api/ideliumcl/testcycle`

These routes are used by `idelium-cli` to create the top-level execution cycle
record and mark it terminal.

## Compatibility contract

`POST /api/ideliumcl/testcycle` accepts:

```json
{
  "testCycleId": 7,
  "executionContext": {
    "environment": "demo",
    "browser": "firefox",
    "browserVersion": "150.0",
    "device": "desktop",
    "deviceName": "local",
    "deviceType": "desktop",
    "platformName": "darwin",
    "platformVersion": "25.0",
    "runtime": "selenium"
  }
}
```

`executionContext` is optional for backward compatibility. When present it must
be a JSON object and is treated as a runtime metadata snapshot, not as a source
of authorization. The Go API redacts sensitive keys before persistence,
including authorization headers, passwords, secrets, tokens, API keys, sessions,
CSRF values, and cookies.

The response remains Laravel-compatible:

```json
{
  "idCycle": 44
}
```

`PUT /api/ideliumcl/testcycle` accepts only terminal status values preserved from
Laravel:

```json
{
  "testCycleId": 44,
  "status": 2
}
```

Missing and cross-tenant references return:

```json
{
  "message": "Invalid details"
}
```

## Tenant isolation

Both routes run behind the legacy CLI API-key middleware. The authenticated
customer ID is the only tenant scope used for writes:

- creation requires the source `test_cycles.id` to belong to the same customer;
- runtime metadata is written only on the tenant-owned performed-cycle row;
- update requires the target `performed_test_cycles.id` to belong to the same
  customer;
- missing and foreign-tenant rows are indistinguishable to callers.

## Runtime metadata snapshots

The CLI sends a bounded `executionContext` object at cycle creation so the Web UI
can show the environment, browser, browser version, operating system, device,
device type, and runtime used for a run.

The current migration keeps the PHP schema freeze intact. Go therefore persists
the redacted snapshot only when the database already exposes one of these
compatible columns on `performed_test_cycles`:

1. `executionContext`
2. `execution_context`
3. `context`

If none of those columns exists, the route still creates the performed cycle
using the legacy Laravel columns and returns the same `idCycle` response. This
keeps existing Docker and Laravel-backed installations compatible while allowing
newer schemas to capture the runtime snapshot immediately.

## Rollback

Rollback is route-level and does not require a database migration:

1. Route `POST /api/ideliumcl/testcycle` and `PUT /api/ideliumcl/testcycle` back
   to Laravel in the gateway or rollout manifest.
2. Keep the same database schema because Go writes the existing Laravel columns:
   `testCycleId`, `date`, `status`, `idCostumer`, `created_at`, and `updated_at`.
   Optional runtime metadata snapshot columns are ignored by Laravel fallback
   unless a later coordinated schema migration explicitly teaches Laravel to
   read them.
3. Re-run the CLI smoke target for performed-cycle writes against the Laravel
   upstream.

No dual-write mode is permitted for this route pair.
