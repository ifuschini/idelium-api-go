# Tenant Isolation Test Helpers

Issue [#95](https://github.com/ifuschini/idelium-api-go/issues/95) adds the
first reusable tenant-isolation test helper package for Go migration slices.
The package lives in `internal/testsupport/tenant` and is intentionally small so
each future repository or handler test can prove ownership without depending on
Laravel internals.

## Purpose

Every tenant-owned Idelium resource must be read or mutated through an ownership
predicate in the same query or transaction that accesses the resource. In other
words, every test must prove the ownership predicate in the same query or
transaction used by production code. The helper package standardizes negative
cross-tenant checks by requiring tests to declare:

- an owner tenant scope;
- a foreign tenant scope;
- the synthetic resource identifier under test;
- a lookup function that uses the production authorization path.

The helper then validates both directions:

1. the owner tenant can still read its own resource;
2. a foreign tenant cannot resolve the owner resource.

## Public helper contract

The package exposes these test-facing primitives:

- `Scope`: synthetic tenant and actor identity for a test case;
- `OwnedRecord`: minimal ownership projection for list-result assertions;
- `LookupFunc`: repository lookup callback used by negative tests;
- `NewScope`: creates validated synthetic scopes;
- `AssertOwnedRecords`: proves list queries only return active-tenant records;
- `AssertOwnerCanRead`: proves the owner success path still works;
- `AssertForeignTenantCannotRead`: proves the negative cross-tenant path;
- `AssertTenantIsolation`: runs the success and denial checks together.

Diagnostics avoid tenant IDs, resource IDs, payloads, credentials, headers, and
cookies. Repository errors are reported by Go type only, which preserves useful
debugging shape while avoiding accidental secret disclosure.

## Adoption rule for future tickets

Any migrated route, repository, or service that touches tenant-owned data must
include tests using these helpers or document why tenant isolation is not
applicable. Database-backed tests must seed at least two synthetic tenants and
verify that cross-tenant identifiers return the Laravel-compatible denial shape,
normally a tenant-scoped `404` for hidden resources or `403` where Laravel
explicitly exposes authorization failure.

## Compatibility and rollback

This change does not move traffic, expose a new HTTP route, add a schema, or
modify Laravel behavior. OpenAPI and Laravel-Go differential comparisons are not
applicable to the helper itself because it is test infrastructure. Rollback is a
normal revert of the helper package and its tests; future migrated tenant-owned
routes must keep equivalent negative coverage before they can remain Go-owned.
