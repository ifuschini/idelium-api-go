# Postman Execution Detail Persistence

## Purpose

Idelium CLI can execute Postman collections through Newman. The Web UI needs the
request-level execution details to show what happened inside a Postman step:
request name, method, URL, status, timing, assertion results, diagnostics, and
safe payload summaries.

## HTTP contract

`PUT /api/ideliumcl/test` remains the write boundary for final performed-test
status. When a Postman test finishes, the CLI may include `postmanData`:

```json
{
  "testId": 55,
  "status": 2,
  "postmanData": [
    {
      "name": "Create user",
      "method": "POST",
      "url": "https://api.example.test/users",
      "time": 123,
      "assertions": [
        {
          "name": "status is 201",
          "passed": true
        }
      ],
      "request": {
        "headers": {
          "Content-Type": "application/json"
        },
        "payload": {
          "username": "demo"
        }
      },
      "response": {
        "status": 201,
        "headers": {
          "Content-Type": "application/json"
        },
        "payload": {
          "id": 42
        }
      }
    }
  ]
}
```

The response remains Laravel-compatible:

```json
{
  "idTest": 55
}
```

## Validation

`postmanData` may be omitted, set to `null`, or sent as a JSON array. Any other
shape is rejected with `422 VALIDATION_FAILED` before database writes occur.

Each array entry is treated as an opaque forward-compatible object because the
Newman schema can evolve. Go does not require a fixed shape, but the documented
fields above are the stable fields consumed by Idelium Web.

## Redaction

The API redacts sensitive keys at any nesting level before persistence:

- authorization headers;
- passwords;
- secrets;
- tokens and API keys;
- session identifiers;
- CSRF values;
- cookies and `Set-Cookie` values.

Non-sensitive request and response payload values are preserved so operators can
debug API behavior without falling back to Newman reports outside Idelium.

## Tenant isolation

The update is allowed only when the target `performed_tests.id` belongs to the
authenticated CLI customer. Missing and foreign-tenant rows return the same
Laravel-compatible `{"message":"Invalid details"}` response.

## Compatibility and rollback

No schema change is required because the existing `performed_tests.postmanData`
column already stores this result payload. Rollback remains route-level:

1. Route `PUT /api/ideliumcl/test` back to Laravel.
2. Keep the database unchanged.
3. Re-run the CLI Postman smoke target and confirm the Web UI can still render
   stored `postmanData`.

No dual-write mode is permitted.
