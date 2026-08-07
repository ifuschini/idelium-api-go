# Idelium API Go Directives

These rules extend the workspace-level Idelium engineering directives.

1. Use English for documentation, comments, diagnostics, and identifiers.
2. Require tenant scope in every tenant-owned repository method and SQL query.
3. Never serialize persistence models directly into HTTP responses.
4. Define externally consumed request and response contracts in OpenAPI first.
5. Keep handlers thin; business rules belong to application/domain services.
6. Validate input and size limits at the HTTP boundary.
7. Make multi-record mutations transactional and retry-safe.
8. Add negative cross-tenant tests for every migrated resource.
9. Pin Go, tools, modules, GitHub Actions, and container images.
10. Run formatting, vet, tests, race tests, and builds before completion.
11. Keep Laravel compatibility deliberate and documented during migration.
12. Never log credentials, authorization headers, cookies, session identifiers,
    environment secrets, or unredacted test payloads.

## Required verification

Run:

```sh
make verify
```

Database-backed changes must also pass the MySQL integration and migration tests.

