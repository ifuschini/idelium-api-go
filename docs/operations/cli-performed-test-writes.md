# CLI Performed Test Writes

## Scope

The Go API now owns the Laravel-compatible CLI performed-test write routes:

- `POST /api/ideliumcl/test`
- `PUT /api/ideliumcl/test`

These routes are used by `idelium-cli` to create a test execution row and to
update the final test status and optional Postman execution detail payload.

## Compatibility contract

`POST /api/ideliumcl/test` accepts:

```json
{
  "testCycleId": 7,
  "testId": 9,
  "name": "Browser test"
}
```

The response remains Laravel-compatible:

```json
{
  "idTest": 55
}
```

`PUT /api/ideliumcl/test` accepts:

```json
{
  "testId": 55,
  "status": 1,
  "postmanData": []
}
```

`postmanData` may be omitted, set to `null`, or provided as an array. Missing or
cross-tenant references return the legacy response:

```json
{
  "message": "Invalid details"
}
```

## Tenant isolation

Both routes run behind the legacy CLI API-key middleware. The authenticated
customer ID is the only tenant scope used for reads and writes:

- creation requires the referenced `performed_test_cycles.id` and `tests.id` to
  belong to the same customer;
- update requires the target `performed_tests.id` to belong to the same customer;
- missing and foreign-tenant rows are indistinguishable to callers.

## Postman payload redaction

When `postmanData` is present, Go validates that it is either `null` or a JSON
array before storing it. Sensitive keys such as authorization headers, cookies,
session identifiers, passwords, secrets, tokens, API keys, and CSRF markers are
replaced with `[REDACTED]`. Request names, methods, URLs, response codes, timing,
and non-sensitive diagnostics remain available for Idelium Web result views.

See [Postman Execution Detail Persistence](postman-execution-detail-persistence.md)
for the stable request-level fields consumed by the Web UI and the redaction
boundary applied before persistence.

## Rollback

Rollback is route-level and does not require a database migration:

1. Route `POST /api/ideliumcl/test` and `PUT /api/ideliumcl/test` back to Laravel
   in the gateway or rollout manifest.
2. Keep the same database schema because Go writes the existing Laravel columns:
   `testCycleDoneId`, `testId`, `name`, `status`, `postmanData`, `idCostumer`,
   `created_at`, and `updated_at`.
3. Re-run the CLI smoke target for performed-test writes against the Laravel
   upstream.

No dual-write mode is permitted for this route pair.
