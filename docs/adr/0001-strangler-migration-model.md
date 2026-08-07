# ADR-0001: Migrate the Laravel API with a route-level strangler model

- Status: Accepted
- Date: 2026-08-07
- Owners: Idelium API maintainers
- Related issues: `#1`, `#15`, `#77`, `#78`
- Supersedes: None
- Superseded by: None

## Context

Idelium Web, CLI, runners, Docker deployments, and persisted MySQL data depend on
the current Laravel API. Replacing the service in one release would combine
contract, authentication, tenancy, persistence, and operational risk into a
single irreversible cutover. Keeping two independent writers active would make
side effects, audit trails, leases, and rollback behavior ambiguous.

The migration requires incremental delivery while the public contract remains
compatible and Laravel stays available as a fallback.

## Decision

Use a route-level strangler model behind the existing Idelium gateway:

1. Laravel initially owns every externally consumed route and every mutation.
2. Go implements a bounded route group without receiving production traffic.
3. OpenAPI, sanitized fixtures, Laravel-Go differential tests, consumer smoke
   tests, tenant-isolation tests, and operational gates prove compatibility.
4. The gateway switches reads only after their evidence passes.
5. Mutation routes move by transaction-owning aggregate. The generated
   [ownership matrix](../contracts/migration-ownership-matrix.md) must show one
   mutation owner for the entire aggregate before and after the switch.
6. Application-level dual writes are prohibited. A request reaches one writer;
   neither runtime replicates an application write into the other.
7. Shared MySQL access is transitional. Schema changes remain backward
   compatible while both runtimes are deployable.
8. Laravel remains the fallback until the migration wave exit criteria and
   rollback observation window are complete.

Route ownership is configuration deployed through the gateway, not a runtime
guess based on errors or availability. Go must not silently proxy an owned
mutation to Laravel after starting a transaction.

## Alternatives considered

### Big-bang replacement

Rejected because it postpones integration feedback and requires simultaneous
validation of every consumer, trust path, and database side effect.

### Application-level dual writes

Rejected because partial failures produce divergent records and make the source
of truth and rollback semantics unclear. Database replication controlled by the
platform is outside this prohibition but cannot be used to assign two
application mutation owners.

### Permanent per-request fallback

Rejected for mutations because retrying a request against a second runtime can
duplicate side effects. Read fallback may be approved for a specific route only
when its timeout and consistency behavior are documented and tested.

### Separate Go database from the first slice

Deferred because data replication and reconciliation would add risk before
contract compatibility is established. Aggregate extraction remains a possible
post-migration decision.

## Consequences

- Migration slices are small, measurable, and independently reversible.
- Laravel and Go must coexist, so shared schemas and deployment artifacts require
  compatibility discipline.
- The gateway becomes a controlled migration component and its route ownership
  configuration requires review and audit evidence.
- Contract and differential test infrastructure is required before meaningful
  traffic migration.
- Aggregate boundaries can make a mutation slice larger than one endpoint, but
  preserve transaction ownership and safe rollback.

## Security and tenant isolation

Each runtime independently authenticates its assigned request and enforces tenant
ownership in the same query or transaction used to access tenant-owned data.
Route switching cannot weaken authorization or translate a denial into fallback
success. Negative cross-tenant tests are mandatory for tenant routes.

Credentials, authorization headers, cookies, session identifiers, customer
payloads, and environment secrets are never copied into logs or golden fixtures.
Only sanitized synthetic fixtures governed by the
[golden fixture policy](../contracts/golden-fixture-policy.md) may be committed.

## Compatibility

The current Laravel-facing status codes, response fields, validation behavior,
authorization semantics, ordering, pagination, side effects, and relevant
headers remain the compatibility baseline unless a separately approved versioned
decision changes them. Web, CLI, runner, Docker, API, and persisted-data changes
must remain backward compatible during coexistence.

This ADR changes no HTTP contract, schema, or runtime behavior. OpenAPI and
Laravel-Go differential coverage are therefore not applicable to the ADR itself;
they are required evidence for each later route switch.

## Deployment and rollback

For each route group:

1. deploy Go dark with readiness, metrics, redacted logs, and compatibility
   evidence;
2. confirm Laravel remains healthy and the ownership matrix still names one
   mutation owner;
3. switch the approved gateway route to Go;
4. observe error rate, latency, authorization denials, tenant isolation,
   side-effect completion, and consumer health against agreed thresholds;
5. stop rollout on threshold breach.

Rollback switches the gateway route back to Laravel before disabling the Go
owner. Because there are no dual writes and coexistence migrations are backward
compatible, rollback does not restore a database snapshot or replay application
writes. Any incompatible schema cleanup occurs only after the Laravel fallback
window has ended through a separate ADR and migration.

## Verification

- The generated ownership matrix covers every production-visible route and
  rejects split mutation ownership.
- Every migrated route requires OpenAPI validation, sanitized golden fixtures,
  Laravel-Go differential coverage, and applicable consumer smoke tests.
- Every tenant-owned route requires success, malformed-input, authorization, and
  negative cross-tenant tests.
- Every route switch requires operational metrics, documented thresholds, and a
  rehearsed gateway rollback.
- `make verify` remains the minimum repository gate; database-backed slices also
  require MySQL integration and migration verification.
