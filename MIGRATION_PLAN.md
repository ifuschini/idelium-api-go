# Idelium API Migration Plan: Laravel/PHP to Go

## 1. Purpose

This document defines the controlled migration of `idelium-api` from Laravel/PHP
to the new `idelium-api-go` repository.

The migration must preserve the public behavior consumed by Idelium Web,
Idelium CLI, Idelium runners, and Docker deployments. It must also preserve
tenant isolation, authentication semantics, database compatibility, result
contracts, and operational rollback throughout the transition.

This is an incremental replacement, not a big-bang rewrite.

## 2. Current baseline

The current Laravel service is not a small CRUD API. At the time this plan was
written, it contains approximately:

- 46 controllers;
- 41 Eloquent models;
- 66 database migrations;
- 159 explicitly declared HTTP routes plus the expanded project resource routes;
- 36 feature-test files and 2 unit-test files;
- 15,267 lines across application, route, and migration PHP sources;
- 8,614 lines of PHP tests.

The existing OpenAPI v1 document covers only a subset of the live routes. It
currently focuses on parallel execution, workload identity, agents, asset
versions, and asset impact. A complete compatibility contract must therefore be
created before route migration begins.

`idelium-api-go` is currently an empty repository with an uncommitted `main`
branch. It is the correct place to establish the new architecture without
carrying Laravel implementation details into the Go package layout.

## 3. Migration goals

1. Preserve the existing Web, CLI, runner, and Docker API contracts during the
   migration.
2. Make tenant ownership mandatory in every query and mutation.
3. Move route ownership in small, reversible domain slices.
4. Keep one authoritative writer for each aggregate at every point in time.
5. Preserve existing data and support upgrades from current Docker databases.
6. Improve startup time, memory use, concurrency, observability, and deployment
   simplicity without changing product behavior accidentally.
7. End with Go owning the API runtime, background jobs, schema migrations, and
   operational documentation.

## 4. Non-goals

- Redesigning the Web application during the backend migration.
- Renaming legacy routes or response fields before compatibility is achieved.
- Replacing MySQL during the language migration.
- Introducing microservices for every domain.
- Rewriting Idelium CLI or the runner.
- Combining schema redesign with route conversion unless a schema defect blocks
  safe migration.

## 5. Target architecture

The first Go release should remain a modular monolith. Domain packages provide
clear ownership boundaries while one deployable service keeps transactions,
operations, and local Docker deployment manageable.

```text
Idelium Web ──────┐
Idelium CLI ──────┼── HTTPS gateway / route switch ──┬── Laravel API
Idelium Runner ───┘                                  └── Go API
                                                           │
                                              ┌────────────┼────────────┐
                                              │            │            │
                                            MySQL       artifacts    job queue
```

During coexistence, the gateway selects the owner by route group. Requests are
never sent to both implementations for mutations. Safe read requests may be
shadowed to Go for comparison, but the shadow response must never be returned to
the client or cause a side effect.

After final cutover:

```text
Idelium Web / CLI / Runner
            │
       HTTPS gateway
            │
       Idelium API Go
            │
   ┌────────┼─────────┐
 MySQL   artifacts   workers
```

## 6. Recommended Go foundation

Use a contract-first, standard-library-compatible stack:

- standard Go modules with all direct and transitive dependencies pinned;
- `net/http` with `chi` for composable route groups and middleware;
- OpenAPI 3.0 as the source of HTTP request and response contracts;
- strict server and model generation with `oapi-codegen`;
- explicit SQL with `sqlc` for type-safe data access;
- the MySQL driver behind `database/sql`;
- versioned SQL migrations with a migration tool pinned in `go.mod`;
- `log/slog` structured JSON logging with centralized redaction;
- OpenTelemetry-compatible traces and metrics;
- native `testing`, table-driven tests, HTTP contract tests, and container-backed
  MySQL integration tests.

All tool versions and container base images must be pinned. Generated OpenAPI and
SQL code should be committed so upgrades remain reviewable and reproducible.

## 7. Proposed repository layout

```text
idelium-api-go/
├── cmd/
│   ├── api/                 # API process
│   ├── worker/              # Background worker process
│   └── migrate/             # Migration entry point
├── api/
│   ├── openapi.yaml         # Complete compatibility contract
│   └── generated/           # Committed generated server/types
├── internal/
│   ├── app/                 # Composition root and lifecycle
│   ├── auth/                # Browser, API key, run token, SSO, MFA
│   ├── tenancy/             # Required tenant context and authorization
│   ├── projects/
│   ├── authoring/           # Tests, steps, cycles, imports, plugins
│   ├── environments/
│   ├── platforms/
│   ├── execution/           # Launcher, runs, workers, results
│   ├── artifacts/
│   ├── identity/
│   ├── integrations/
│   ├── audit/
│   ├── persistence/         # sqlc queries and transaction helpers
│   ├── httpx/               # Errors, pagination, correlation, recovery
│   └── observability/
├── db/
│   ├── migrations/
│   ├── queries/
│   └── schema/
├── test/
│   ├── contract/            # Laravel versus Go golden comparisons
│   ├── integration/
│   ├── tenancy/
│   └── fixtures/
├── deployments/
│   └── docker/
├── docs/
│   ├── architecture/
│   ├── compatibility/
│   └── operations/
├── scripts/
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile
└── MIGRATION_PLAN.md
```

Package boundaries must follow business capabilities, not Laravel controller or
Eloquent model names. HTTP handlers should call application services; application
services should depend on repository interfaces; SQL implementations should stay
inside persistence packages.

## 8. Mandatory engineering rules

### 8.1 Tenant isolation

- Resolve an immutable tenant context in authentication middleware.
- Require `tenantID` as an argument for every tenant-owned repository operation.
- Include tenant ownership in the same SQL statement used to read, update, or
  delete a resource.
- Never load a resource globally and check its tenant afterward.
- Return the same not-found behavior for missing and cross-tenant resources when
  disclosure would leak existence.
- Add negative integration tests for every migrated resource: customer A cannot
  read, update, delete, relate, export, or execute customer B's data.
- Prevent unscoped SQL queries through code review checks and repository APIs.

### 8.2 Response safety

- Define explicit response DTOs in OpenAPI.
- Never serialize database rows directly.
- Redact passwords, API keys, tokens, cookies, environment secrets,
  authorization headers, and sensitive Postman payload fields.
- Preserve the existing stable error envelope and HTTP status behavior.

### 8.3 Transactions and idempotency

- Use transactions for imports, ordering, cycle creation, result persistence,
  credential rotation, and multi-table deletes.
- Define idempotency keys for runner claims, heartbeats, result writes, export
  requests, and integration delivery replay.
- Preserve uniqueness and foreign-key enforcement in MySQL.

## 9. Contract-first preparation

Before implementing business routes in Go:

1. Export the complete Laravel route list.
2. Inventory every consumer in `idelium-web`, `idelium-cli`, Docker scripts, and
   runner workflows.
3. Expand `openapi-v1.yaml` to every externally consumed route.
4. Record for each operation:
   - authentication mode;
   - required capability or role;
   - tenant ownership rule;
   - request schema and limits;
   - response schema and optional fields;
   - pagination, filtering, and sorting behavior;
   - error codes and status codes;
   - transaction and idempotency behavior;
   - side effects and audit events.
5. Capture sanitized golden fixtures from the Laravel implementation.
6. Generate clients used by the compatibility suite.

The compatibility suite must compare semantic JSON, headers, status codes, and
side effects. It should normalize only nondeterministic fields such as timestamps,
UUIDs, ordering explicitly declared as irrelevant, and correlation identifiers.

## 10. Database migration strategy

### 10.1 Coexistence period

- Keep MySQL as the shared persistence layer.
- Keep Laravel as the only schema-migration owner until the schema handover
  milestone.
- Give each route group exactly one mutation owner: Laravel or Go.
- Do not implement application-level dual writes.
- Add only backward-compatible schema changes while both runtimes are active.
- Mirror the effective MySQL schema into Go integration-test fixtures and run
  `sqlc` validation in CI.
- Test Go against both an empty database initialized by Laravel migrations and a
  sanitized upgrade snapshot from the previous supported release.

Sharing the database is a temporary migration technique, not the final domain
boundary. It is acceptable only while route ownership and transaction ownership
are explicit.

### 10.2 Schema ownership handover

1. Freeze Laravel schema changes.
2. Produce a reviewed Go baseline migration representing the complete supported
   schema.
3. Add a bridge command that detects a fully migrated Laravel database and marks
   the Go baseline as applied without recreating tables.
4. Verify new installations from an empty database using Go migrations.
5. Verify in-place upgrades from the last Laravel-owned release.
6. Transfer migration ownership to `idelium-api-go`.
7. Reject startup when the database schema is newer or older than the supported
   compatibility window.

Rollback after schema handover must remain possible for at least one release.
Only additive changes should be used during that release window.

## 11. Authentication migration strategy

The API currently has several distinct trust paths and they must not be collapsed
into one generic token check:

- browser authentication with Laravel Sanctum, sessions, cookies, and CSRF;
- legacy customer API keys;
- scoped service-account credentials;
- runner and parallel-run tokens;
- OIDC workload identity exchange;
- SSO callbacks, SCIM lifecycle, MFA, and break-glass controls.

### Temporary browser-auth bridge

Browser authentication should remain on Laravel until the stateless surfaces are
stable. During the intermediate phase, a Go endpoint that needs a browser identity
may call an internal Laravel introspection endpoint over the private Docker
network. The bridge must:

- accept no public traffic;
- authenticate the Go service with a rotated internal credential or mTLS;
- return only user ID, tenant ID, role, capabilities, session expiry, and step-up
  state;
- never return session cookies, password hashes, or API keys;
- use short timeouts and fail closed;
- avoid caching authorization longer than the session's remaining lifetime.

### Final browser-auth ownership

Go must eventually implement the existing frontend-visible login, logout, cookie,
CSRF, expiry, customer-switching, and 401/419 behavior. A browser compatibility
suite must prove that the current Web build works without source changes before
the auth routes move to Go.

Credential hashing and comparison formats must remain compatible during the
transition. Any stronger replacement format needs a versioned, opportunistic
rehash strategy.

## 12. Migration waves

### Wave 0 — Baseline and governance

Deliverables:

- repository directives, Apache 2.0 license, README, contribution guide, and ADR
  template;
- full route and consumer inventory;
- complete OpenAPI compatibility backlog;
- sanitized database fixtures and golden responses;
- agreed SLOs, rollout metrics, and rollback thresholds;
- ownership matrix for Laravel and Go routes.

Exit criteria:

- every public route has an owner and compatibility record;
- hidden or undocumented consumer dependencies are recorded;
- no implementation begins without a contract test target.

### Wave 1 — Go service skeleton

Deliverables:

- API and worker binaries;
- configuration loader with startup validation;
- graceful shutdown and bounded HTTP timeouts;
- health, readiness, and build-information endpoints;
- request correlation, structured logging, panic recovery, CORS, and response
  redaction;
- MySQL connection management and migration-status checks;
- CI for format, vet, static analysis, unit tests, race tests, integration tests,
  image build, SBOM, and vulnerability scanning;
- pinned multi-stage distroless or minimal runtime container.

Exit criteria:

- the empty service deploys beside Laravel in Docker;
- readiness fails when required dependencies or schema versions are invalid;
- the image runs as non-root and handles termination cleanly.

### Wave 2 — Contract and differential test harness

Deliverables:

- generated OpenAPI server interfaces and compatibility clients;
- golden request/response runner against Laravel and Go;
- database side-effect comparator;
- tenant-isolation test library;
- Web and CLI smoke suites able to target either backend;
- performance baseline for representative reads and writes.

Exit criteria:

- CI can fail on status, payload, header, authorization, or side-effect drift;
- sensitive fields are removed from captured fixtures.

### Wave 3 — Low-risk stateless reads

Suggested first domains:

- status, capabilities metadata, and safe build information;
- platform catalog reads: types, statuses, operating systems, OS versions,
  browsers, browser versions, brands, models, and locations;
- other read-only lookups with no browser-session dependency.

Rollout:

- shadow safe GET requests to Go;
- compare responses and latency;
- route a small percentage to Go;
- move each route permanently only after parity gates pass.

### Wave 4 — CLI configuration reads

Migrate the `ideliumcl` read paths:

- test cycle;
- test;
- step;
- plugin and plugin list;
- environment and environment list.

This wave uses API-key authentication, making it independent from browser session
compatibility. It must preserve missing-reference diagnostics and tenant-scoped
404 behavior expected by Idelium CLI.

Exit criteria:

- the supported CLI release passes its complete remote execution suite against
  Go;
- Laravel and Go return equivalent configuration graphs for the same cycle;
- cross-tenant identifiers never resolve.

### Wave 5 — CLI result writes and execution lifecycle

Migrate:

- performed cycle creation and update;
- performed test creation and update;
- performed step creation and update;
- Postman execution detail persistence;
- runtime metadata, browser, OS, device, environment, and platform snapshots;
- failure finalization so runs cannot remain incorrectly pending.

Exit criteria:

- writes are transactional and idempotent;
- partial network failures can be retried safely;
- Web displays identical step-by-step results produced through Go;
- result redaction and artifact size policies pass contract tests.

### Wave 6 — Authoring and project resources

Migrate bounded contexts in this order:

1. projects;
2. environments and secret policy;
3. plugins and plugin manifest validation;
4. steps, ordering, JSON, wizard, and DSL payloads;
5. tests and step membership;
6. test cycles and test ordering;
7. test import and launcher.

Each context moves as a complete read/write unit. Do not split one aggregate's
writes between Laravel and Go.

Exit criteria:

- all validation limits and error codes match the contract;
- imports and ordering operations are transactional;
- Web authoring workflows pass end-to-end tests;
- DSL and Postman payloads retain byte-safe and schema-safe behavior.

### Wave 7 — Results, artifacts, grid, and integrations

Migrate:

- performed cycle/test/step exploration;
- result exports and downloads;
- artifact descriptors, retention, archive, restore, legal hold, and deletion;
- enterprise grid query snapshots and bulk jobs;
- integration endpoints, deliveries, retries, and secret rotation;
- audit event reads and writes;
- asset impact and asset version review workflows.

Background jobs should move with their owning domain. A job created by Go must be
consumed by a Go worker using a versioned payload. Existing Laravel jobs must drain
before queue ownership changes.

### Wave 8 — Parallel runs, agents, and runner API

Migrate:

- schedules and matrices;
- worker claims, heartbeats, updates, cancellation, and results;
- run-token issue, validation, and revocation;
- agent registration and status;
- runner-only endpoints.

This is a high-concurrency wave. It requires row-locking review, lease expiry,
idempotent claims, clock-skew tests, race tests, load tests, and failure injection.

Exit criteria:

- no run can be claimed twice;
- stale workers are detected deterministically;
- cancellation and finalization converge under retries;
- Go passes race tests and the parallel execution integration suite.

### Wave 9 — Browser authentication and administration

Migrate last:

- login, logout, CSRF, sessions, and current-user endpoints;
- menu/header customer switching and capability resolution;
- accounts, roles, profiles, and password policy;
- customer administration and legacy API-key lifecycle;
- service accounts;
- MFA and step-up authentication;
- SSO, OIDC callbacks, SCIM, workload identity, and break-glass controls.

Exit criteria:

- the unmodified Web client passes login, reload, logout, expiry, customer switch,
  CORS, and CSRF tests;
- negative authorization and tenant-isolation suites pass for every role;
- credential rotation and last-used timestamps remain correct;
- the temporary auth bridge is disabled and removed.

### Wave 10 — Schema handover and final cutover

Deliverables:

- Go-owned baseline and incremental migrations;
- empty-install and upgrade-path verification;
- 100% Go traffic in staging, then canary production traffic;
- Laravel queue drain and write freeze;
- rollback rehearsal;
- Docker default changed to the Go image;
- updated operations, backup, recovery, and release documentation.

Exit criteria:

- no production route depends on Laravel;
- all compatibility, security, load, and recovery gates pass;
- rollback to the last dual-runtime release has been rehearsed;
- Laravel enters a time-boxed read-only maintenance period before archival.

## 13. Traffic migration and rollback

`idelium-docker` should add an internal route switch with configuration such as:

```text
IDELIUM_API_OWNER_PLATFORM_CATALOG=go
IDELIUM_API_OWNER_IDELIUMCL_READ=go
IDELIUM_API_OWNER_AUTHORING=laravel
```

The exact implementation may be gateway route maps rather than environment
variables, but ownership must be visible and auditable.

For each wave:

1. deploy Go with the route disabled;
2. run direct contract tests against Go;
3. shadow safe reads;
4. enable a staging route;
5. enable a production canary;
6. monitor error, latency, authorization-denial, and database metrics;
7. move to full traffic;
8. keep the previous route owner deployable until the rollback window closes.

Rollback means changing route ownership back to Laravel. It must not require a
database restore. This is why coexistence migrations must remain additive.

Immediate rollback triggers include:

- cross-tenant access or credential disclosure;
- contract-breaking responses;
- elevated authentication failures;
- non-idempotent duplicate writes;
- data corruption or orphaned relationships;
- sustained latency or error-rate breach;
- runs left in an incorrect non-terminal state.

## 14. CI and quality gates

Every pull request in `idelium-api-go` should run:

```text
format -> vet -> static analysis -> unit -> integration -> tenancy
       -> OpenAPI compatibility -> race -> build -> image/security checks
```

Required gates:

- `gofmt` and import formatting;
- `go vet` and pinned static analysis;
- unit tests with coverage reporting;
- MySQL integration tests;
- tenant-isolation negative tests;
- OpenAPI request and response validation;
- Laravel-versus-Go golden contract tests for migrated routes;
- `go test -race` for concurrency-sensitive packages;
- migration tests from empty and previous schemas;
- Web and CLI cross-repository smoke tests;
- dependency, container, license, secret, and vulnerability scanning;
- deterministic container build with provenance and SBOM.

Generated sources must be checked for drift in CI.

## 15. Observability and performance gates

Record a Laravel baseline before optimization claims are made. Compare at least:

- p50, p95, and p99 request latency;
- throughput and error rate;
- startup and readiness time;
- resident memory and CPU under idle and load;
- MySQL query count and duration per endpoint;
- connection-pool saturation;
- queue depth and job duration;
- run claim and heartbeat convergence;
- authorization denials by reason without sensitive identifiers.

Logs must include correlation ID, route, status, duration, and safe tenant/user
surrogates. They must never include credentials, authorization headers, session
IDs, raw environment secrets, or unredacted test payloads.

## 16. Security review checklist

- Threat-model every authentication path before migration.
- Enforce request body, collection, upload, artifact, and export size limits.
- Configure HTTP header, body, read, write, idle, and shutdown timeouts.
- Use constant-time credential comparisons where applicable.
- Store only credential hashes when the protocol permits it.
- Rotate internal bridge credentials and remove them after auth cutover.
- Trust proxy headers only from configured proxy networks.
- Validate outbound integration destinations against SSRF policy.
- Preserve audit events for credential, identity, tenant, and destructive actions.
- Fuzz parsers and complex import/result payload boundaries.
- Run an independent tenant-isolation review before each production wave.

## 17. Delivery estimate

For two experienced backend engineers plus shared Web/CLI/DevOps support, a
realistic initial range is 20 to 30 calendar weeks. The range must be recalibrated
after Waves 0 through 2 because the undocumented contract surface is the largest
uncertainty.

| Workstream | Indicative effort |
| --- | ---: |
| Baseline, full contract, and differential harness | 4-6 weeks |
| Go foundation and low-risk reads | 3-4 weeks |
| CLI reads and result writes | 3-5 weeks |
| Authoring and project resources | 4-6 weeks |
| Results, artifacts, grid, and integrations | 4-6 weeks |
| Parallel execution and agents | 3-5 weeks |
| Authentication, identity, and administration | 4-6 weeks |
| Schema handover, cutover, and stabilization | 2-4 weeks |

Several workstreams can overlap after the contract harness is stable, but route
cutovers must follow the dependency order above.

## 18. Initial backlog for `idelium-api-go`

Create these issues first:

1. **Epic: Complete API contract and consumer inventory**
2. **Epic: Bootstrap production-grade Go service**
3. **Epic: Build Laravel-Go differential contract harness**
4. **Epic: Implement tenant context and scoped persistence foundation**
5. **Epic: Migrate platform catalog reads**
6. **Epic: Migrate Idelium CLI configuration reads**
7. **Epic: Migrate CLI result persistence and finalization**
8. **Epic: Migrate authoring aggregates**
9. **Epic: Migrate result, artifact, grid, and integration domains**
10. **Epic: Migrate parallel execution and runner APIs**
11. **Epic: Migrate browser authentication and identity lifecycle**
12. **Epic: Transfer schema ownership and cut over Docker**

The first implementation tickets should be:

1. Add workspace directives, license, README, and ADR template.
2. Export and classify every Laravel route.
3. Map every Web, CLI, runner, and Docker consumer to route operations.
4. Define the stable error envelope, pagination, and correlation headers.
5. Scaffold API/worker/migrate binaries with graceful lifecycle management.
6. Add pinned CI, static analysis, race tests, and container build.
7. Add MySQL integration-test infrastructure.
8. Implement tenant context and prove cross-tenant denial with negative tests.
9. Add OpenAPI generation and generated-code drift checks.
10. Add the Laravel-Go golden response comparator.
11. Implement readiness and build-information endpoints.
12. Migrate one read-only platform endpoint as the reference vertical slice.

## 19. Definition of done for each migrated operation

An operation is not migrated merely because its happy path works. It is complete
only when:

- its OpenAPI contract is complete and reviewed;
- request validation, limits, and error behavior match;
- tenant scope and capability checks are enforced in the data query;
- explicit response fields contain no sensitive data;
- happy-path, negative, cross-tenant, and malformed-input tests pass;
- Laravel-Go differential tests pass;
- Web, CLI, or runner consumer tests pass as applicable;
- transaction, idempotency, and audit behavior are verified;
- metrics, logs, and traces are present and redacted;
- load and timeout expectations pass;
- the gateway route can be switched back without restoring the database;
- English compatibility and operations documentation is updated.

## 20. First decision checkpoint

Do not commit to a final delivery date until Waves 0 through 2 are complete. At
that checkpoint, review:

- the number of undocumented response variants;
- Laravel behaviors that must be preserved versus intentionally deprecated;
- browser session bridge feasibility;
- query and schema defects discovered by tenant-scoped SQL;
- parallel-run concurrency requirements;
- performance baselines;
- staffing available for Web, CLI, Docker, security, and migration testing.

The recommended first product milestone is not “all APIs in Go.” It is:

> Idelium CLI can read a complete tenant-scoped test configuration from Go, while
> Laravel remains available for immediate route rollback and the compatibility
> suite proves equivalent behavior.

That milestone validates the architecture, authentication, persistence, tenant
isolation, contract tooling, Docker routing, and rollback model before the most
risky domains are moved.
