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
  "testCycleId": 7
}
```

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
- update requires the target `performed_test_cycles.id` to belong to the same
  customer;
- missing and foreign-tenant rows are indistinguishable to callers.

## Rollback

Rollback is route-level and does not require a database migration:

1. Route `POST /api/ideliumcl/testcycle` and `PUT /api/ideliumcl/testcycle` back
   to Laravel in the gateway or rollout manifest.
2. Keep the same database schema because Go writes the existing Laravel columns:
   `testCycleId`, `date`, `status`, `idCostumer`, `created_at`, and `updated_at`.
3. Re-run the CLI smoke target for performed-cycle writes against the Laravel
   upstream.

No dual-write mode is permitted for this route pair.
