# CLI Performed Step Writes

## Scope

The Go API now owns the Laravel-compatible CLI performed-step write routes:

- `POST /api/ideliumcl/step`
- `PUT /api/ideliumcl/step`

These routes are used by `idelium-cli` to create step execution rows, store
versioned result payloads, and update screenshot artifact metadata for completed
or failed steps.

## Compatibility contract

`POST /api/ideliumcl/step` accepts the legacy CLI payload:

```json
{
  "testCycleId": 44,
  "testId": 55,
  "stepId": 12,
  "name": "Open page",
  "status": 1,
  "screenshots": "[]",
  "data": "{\"result\":\"ok\"}",
  "type": "selenium"
}
```

The response remains Laravel-compatible:

```json
{
  "idStep": 77
}
```

`type` is constrained to the current CLI step runtimes:

- `selenium`
- `seleniumOrAppium`
- `postman`
- `dsl`

`PUT /api/ideliumcl/step` accepts:

```json
{
  "stepId": 77,
  "screenshots": "[]"
}
```

Missing or cross-tenant references return the legacy response:

```json
{
  "message": "Invalid details"
}
```

## Tenant isolation

Both routes run behind the legacy CLI API-key middleware. The authenticated
customer ID is the only tenant scope used for reads and writes:

- creation requires the referenced `performed_test_cycles.id`,
  `performed_tests.id`, and source `steps.id` to belong to the same customer;
- the performed test must belong to the same performed cycle supplied in the
  request;
- update requires the target `performed_steps.id` to belong to the same
  customer;
- missing and foreign-tenant rows are intentionally indistinguishable to callers.

## Result payload validation and redaction

`data` and `screenshots` must be JSON strings on create, matching the existing
Laravel CLI contract. Go validates `data` before persistence and redacts
sensitive keys and credential-like values with `[REDACTED]`. This keeps Idelium
Web result views useful while preventing tokens, cookies, authorization headers,
passwords, CSRF markers, session IDs, and API keys from being stored or returned.

## Rollback

Rollback is route-level and does not require a database migration:

1. Route `POST /api/ideliumcl/step` and `PUT /api/ideliumcl/step` back to
   Laravel in the gateway or rollout manifest.
2. Keep the same database schema because Go writes the existing Laravel columns:
   `testCycleDoneId`, `testDoneId`, `stepId`, `name`, `status`, `screenshots`,
   `type`, `data`, `idCostumer`, `created_at`, and `updated_at`.
3. Re-run the CLI smoke target for performed-step writes against the Laravel
   upstream.

No dual-write mode is permitted for this route pair.
