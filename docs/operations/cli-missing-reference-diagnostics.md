# CLI Missing-Reference Diagnostics

Wave 4 preserves the legacy Idelium CLI contract for missing, malformed, and
cross-tenant configuration references while the read graph moves from Laravel to
Go.

## Public behavior

Go-owned CLI configuration reads keep the Laravel-compatible response:

```json
{"message":"Invalid id"}
```

The response is returned with HTTP `404` when:

- the path identifier is malformed or non-positive;
- a single-resource identifier does not exist;
- a single-resource identifier belongs to another customer tenant;
- the graph references a resource that cannot be resolved through a
  tenant-scoped lookup.

List reads preserve Laravel empty-list behavior for valid tenant-owned projects:
an empty project returns `[]`, not `404`.

## Tenant safety

Missing and cross-tenant resources intentionally produce the same public
diagnostic. This prevents tenant enumeration and preserves the current CLI
behavior. Repository lookups enforce `idCostumer` in the same SQL query that
loads the row or list.

## Observability

Server logs may include the route path, correlation id, and safe failure class.
They must not include:

- `Idelium-Key` values;
- passwords or API tokens;
- session identifiers;
- database connection strings;
- raw database error text that can contain sensitive values.

## Verification

The behavior is covered by:

- handler tests for malformed, missing, and cross-tenant reads;
- router tests for legacy API-key protected `404 {"message":"Invalid id"}`
  behavior;
- OpenAPI diagnostics tests requiring `LegacyMessageResponse` for all Go-owned
  CLI read `404` responses;
- MySQL integration tests that hide cross-tenant rows for every migrated CLI
  graph resource.
