# Artifact lifecycle cutover

Issue #144 moves artifact lifecycle management to Go ownership for the
browser-session routes that change descriptor state or legal-hold metadata.

| Public path | Go path | Behavior |
| --- | --- | --- |
| `POST /api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | `/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | Sets state to `archived` and stores archive metadata |
| `POST /api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | `/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | Sets state to `deleted` unless legal hold blocks deletion |
| `PUT /api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | `/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | Updates `metadata.legalHold` |
| `POST /api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | `/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | Restores archived descriptors to `available` |

## Compatibility decision

The route payloads, status codes, response envelope, and validation messages
match the Laravel controller and `ArtifactLifecycleService` behavior. Go uses
`artifacts.manage` for every mutating route and returns `404` for missing or
cross-tenant descriptors.

Go records append-only `audit_events` rows for successful legal-hold, archive,
restore, and delete-marker changes with the Laravel action names:
`artifact.legal_hold`, `artifact.archive`, `artifact.restore`, and
`artifact.mark_deleted`.

## Tenant isolation and safe diagnostics

Every lifecycle mutation runs in a transaction and locks the descriptor with
`id`, `idCostumer`, `idProject`, and `performedTestCycleId` in the same query.
Audit records use the actor tenant, active tenant, project id, source IP, and
correlation id without recording session identifiers or authorization headers.

Metadata is read raw for mutation safety, then responses and audit snapshots
redact sensitive keys such as authorization, cookies, sessions, API keys,
tokens, passwords, and secrets.

## Deployment and rollback

Before route cutover, run browser handler tests, MySQL lifecycle integration,
Laravel `ArtifactDescriptorTest` in pinned PHP Docker, and staging smoke for:

- legal hold enable and disable;
- legal-hold blocked delete marker;
- archive with reason and restoreBy;
- restore from archived state;
- cross-tenant `404`;
- missing `artifacts.manage` `403`.

Rollback is route-level: switch the four lifecycle routes back to Laravel. The
Go implementation writes only the existing `artifact_descriptors` and
`audit_events` tables and does not introduce a schema migration or dual writes.
