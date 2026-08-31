# Artifact descriptor cutover

Issue #143 moves browser-session artifact descriptor reads to Go ownership and
adds the matching Go repository write path used by artifact-producing services.

| Public path | Go path | Response |
| --- | --- | --- |
| `GET /api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | `/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | Returns `{"data":[...]}` ordered by `artifactType`, then `name` |
| `GET /api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | `/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | Returns `{"data":{...}}` for a tenant-owned descriptor |

## Compatibility decision

Laravel does not expose a public create route for artifact descriptors; writes
are performed through `ArtifactDescriptorService::register`. Go therefore adds
an internal repository write method instead of introducing a new HTTP route.

Descriptor responses preserve the Laravel field names and legacy `idCostumer`
spelling: `id`, `idCostumer`, `idProject`, `performedTestCycleId`,
`performedTestId`, `performedStepId`, `artifactType`, `name`, `contentType`,
`sizeBytes`, `checksumSha256`, `storageKey`, `state`, `retentionUntil`,
`metadata`, `created_at`, and `updated_at`.

## Tenant isolation and safe diagnostics

Every read scopes descriptors by the active Go browser-session tenant, project,
and performed test cycle in the same query. Single-descriptor reads additionally
match the descriptor id in the same predicate, so foreign descriptors are hidden
as not found.

Repository writes first load the performed test cycle by id and active tenant
before inserting into `artifact_descriptors`, normalize SHA-256 checksums to
lowercase, default `state` to `available`, and default retention to 30 days when
callers omit it. Metadata is stored as supplied after JSON validation, while
responses redact sensitive metadata keys such as authorization, cookies,
sessions, API keys, tokens, passwords, and secrets.

## Deployment and rollback

Before route cutover, run the browser handler tests, MySQL-backed integration
test, OpenAPI sync check, and staging smoke for list, show, cross-tenant `404`,
and missing capability `403`.

Rollback is route-level: switch the two descriptor read routes back to Laravel.
The write path uses the existing Laravel-owned `artifact_descriptors` table and
does not introduce dual writes or a schema migration, so rollback does not
require data migration.
