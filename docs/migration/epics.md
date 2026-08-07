# Idelium API Go Migration Epics

This backlog translates `MIGRATION_PLAN.md` into executable GitHub epics,
tracks, and tickets.

The hierarchy is intentionally shallow:

- **Wave epic**: one migration phase with a clear rollout boundary.
- **Domain track**: a functional slice inside a wave.
- **Implementation ticket**: a reviewable change that can be tested and
  committed independently.

Avoid adding deeper nesting unless a track becomes too large to review safely.
Use labels for cross-cutting concerns such as `security`, `tenant-isolation`,
`contract`, `docker`, `web-smoke`, `cli-smoke`, and `rollback`.

## Completion Policy

Each completed migration-plan step must end with a dedicated commit. A ticket is
complete only when its contract, tests, documentation, and rollback notes are
updated as needed.

## Current Status

| Area | Status | Evidence |
| --- | --- | --- |
| Wave 1 foundation | Done | `e4e5def feat: bootstrap Go API foundation` |
| Wave 3 first vertical slice | Done | `d4b7f22 feat: add read-only platform catalogs` |
| GitHub issue materialization | Done | 11 epics, 60 tracks, and 107 tickets mapped in `github-issues.json`. |

## Wave 0 Epic: Baseline And Governance

Goal: make the Laravel-to-Go migration observable, reviewable, and reversible
before additional route ownership moves.

Tracks:

- Route and consumer inventory.
- Compatibility contract backlog.
- Golden fixture strategy.
- Rollout, ownership, and rollback governance.

Tickets:

- Export every Laravel route with method, path, controller, authentication mode,
  and current owner.
- Map Idelium Web, Idelium CLI, runners, Docker scripts, and wiki examples to
  the routes they consume.
- Classify every route as browser session, API key, run token, internal service,
  or public operational endpoint.
- Define the migration ownership matrix with one mutation owner per aggregate.
- Establish the initial compatibility-contract backlog for every inventoried
  public route.
- Create the initial sanitized golden fixture policy.
- Add an ADR template and write the first ADR for the strangler migration model.
- Define release, rollback, and route-switch approval gates.

Exit criteria:

- Every public route has an owner, consumer record, and migration target wave.
- Undocumented consumers are recorded before implementation begins.
- No new route moves without a contract-test target.

## Wave 1 Epic: Production-Grade Go Service Foundation

Goal: run a hardened empty Go service beside Laravel.

Tracks:

- Runtime lifecycle.
- Configuration and secret loading.
- HTTP safety and observability.
- MySQL connectivity.
- CI and container build.

Tickets:

- Create repository directives, README, Apache 2.0 license, Makefile, and
  pinned Dockerfile.
- Scaffold the API process with graceful shutdown, bounded timeouts, and build
  metadata.
- Add health and readiness endpoints with stable response contracts.
- Add structured request logging, correlation IDs, panic recovery, and secure
  headers.
- Add database configuration with secret-file support and redacted failures.
- Add MySQL readiness and integration-test infrastructure.
- Add CI gates for formatting, vet, unit tests, race tests, integration tests,
  and image build.
- Add SBOM and vulnerability scanning gates.
- Add worker and migrate process skeletons when their first owning domains are
  scheduled.

Exit criteria:

- The service deploys beside Laravel.
- Readiness fails closed when dependencies are unavailable.
- The runtime image is pinned, non-root, and reproducible.

## Wave 2 Epic: Contract And Differential Harness

Goal: prove Go and Laravel behavior are equivalent before route traffic moves.

Tracks:

- OpenAPI generation.
- Laravel-Go golden comparison.
- Tenant isolation test library.
- Web and CLI smoke targeting.
- Side-effect comparison.

Tickets:

- Expand OpenAPI to include every externally consumed Laravel route.
- Add generated server interfaces and drift checks.
- Build a golden request/response comparator for safe reads.
- Build a database side-effect comparator for mutations.
- Normalize nondeterministic fields such as timestamps, UUIDs, and correlation
  IDs.
- Add fixture redaction for credentials, session IDs, cookies, authorization
  headers, and test payload secrets.
- Add tenant-isolation test helpers for cross-tenant negative checks.
- Add Web smoke tests that can target Laravel or Go by route owner.
- Add CLI smoke tests that can target Laravel or Go by route owner.
- Add a performance baseline report for representative reads and writes.

Exit criteria:

- CI fails on status, payload, header, authorization, or side-effect drift.
- Sensitive fields are never persisted in captured fixtures.

## Wave 3 Epic: Low-Risk Stateless Reads

Goal: move read-only lookups that do not require browser-session compatibility.

Tracks:

- Operational metadata.
- Platform catalog reads.
- Safe lookup reads.
- Shadow traffic and route switching.

Tickets:

- Migrate platform type reads.
- Migrate platform status reads.
- Migrate location reads.
- Migrate brand reads.
- Migrate model reads.
- Migrate operating-system reads.
- Migrate OS-version reads.
- Migrate browser reads.
- Migrate browser-version reads.
- Define and migrate any remaining safe lookup reads discovered by the route
  inventory.
- Add Laravel-Go differential tests for each catalog endpoint.
- Add gateway route ownership configuration for platform catalog reads.
- Add shadow-read comparison for safe GET requests.

Exit criteria:

- Every migrated read has OpenAPI, unit, integration, and differential tests.
- Route ownership can return to Laravel without database restore.

## Wave 4 Epic: Idelium CLI Configuration Reads

Goal: let Idelium CLI read complete execution configuration from Go.

Tracks:

- API-key authentication.
- Test cycle graph reads.
- Test, step, plugin, and environment reads.
- Missing-reference diagnostics.
- Cross-tenant denial.

Tickets:

- Implement legacy customer API-key authentication with redacted diagnostics.
- Migrate test-cycle read endpoints.
- Migrate test read endpoints.
- Migrate step read endpoints.
- Migrate plugin and plugin-list read endpoints.
- Migrate environment and environment-list read endpoints.
- Preserve CLI missing-reference diagnostics and tenant-scoped 404 behavior.
- Add cross-tenant denial tests for every CLI configuration graph resource.
- Add complete remote CLI execution smoke tests against Go.
- Add Laravel-Go graph equivalence tests for the same cycle.

Exit criteria:

- The supported CLI release can execute remote configurations read from Go.
- Cross-tenant identifiers never resolve.

## Wave 5 Epic: CLI Result Writes And Execution Lifecycle

Goal: persist CLI executions, step results, Postman detail, runtime metadata,
and final states through Go.

Tracks:

- Performed cycle lifecycle.
- Performed test lifecycle.
- Performed step lifecycle.
- Runtime metadata snapshots.
- Postman execution detail.
- Failure finalization.

Tickets:

- Migrate performed-cycle creation and update.
- Migrate performed-test creation and update.
- Migrate performed-step creation and update.
- Persist environment, platform, browser, operating system, and device snapshots.
- Persist Postman request-level details, payload summaries, assertions, and
  response metadata with redaction.
- Enforce artifact size and storage policy.
- Add idempotency keys for retries.
- Ensure failed or interrupted CLI runs always reach a terminal state.
- Add Web result-display parity tests for Go-produced executions.

Exit criteria:

- Writes are transactional and retry-safe.
- Failed network calls do not leave runs pending.
- Web displays the same step-by-step results produced through Go.

## Wave 6 Epic: Authoring And Project Resources

Goal: migrate Web authoring aggregates without splitting write ownership.

Tracks:

- Projects.
- Environments.
- Plugins.
- Steps, JSON, wizard, and DSL.
- Tests and step membership.
- Test cycles and ordering.
- Imports and launcher setup.

Tickets:

- Migrate project reads and writes.
- Migrate environment reads and writes with secret policy.
- Migrate plugin reads, writes, and manifest validation.
- Migrate step reads, writes, ordering, JSON, wizard, and DSL payload handling.
- Migrate test reads, writes, and step membership.
- Migrate test-cycle reads, writes, and test ordering.
- Migrate Idelium JSON import.
- Migrate Postman collection import.
- Migrate launcher configuration reads.
- Add end-to-end Web authoring workflow tests.

Exit criteria:

- Each aggregate has one write owner.
- Imports and ordering operations are transactional.
- DSL and Postman payloads remain byte-safe and schema-safe.

## Wave 7 Epic: Results, Artifacts, Grid, And Integrations

Goal: move analytical, artifact, export, and integration domains to Go.

Tracks:

- Performed result exploration.
- Exports and downloads.
- Artifact lifecycle.
- Grid query snapshots.
- Integrations and deliveries.
- Audit and asset review workflows.

Tickets:

- Migrate performed cycle, test, and step exploration APIs.
- Migrate result export creation, status, and download.
- Migrate artifact descriptor reads and writes.
- Migrate retention, archive, restore, legal hold, and deletion workflows.
- Migrate enterprise grid query snapshots and bulk jobs.
- Migrate integration endpoints, deliveries, retry, and secret rotation.
- Migrate audit event reads and writes.
- Migrate asset impact and asset version review workflows.
- Drain Laravel jobs before moving queue ownership.

Exit criteria:

- Jobs move with their owning domain.
- Versioned Go job payloads are used for Go-created jobs.
- Retention and legal-hold rules remain enforceable.

## Wave 8 Epic: Parallel Runs, Agents, And Runner API

Goal: migrate high-concurrency execution coordination safely.

Tracks:

- Schedules and matrices.
- Worker claims and leases.
- Heartbeats and cancellation.
- Run tokens.
- Agents.
- Runner-only endpoints.

Tickets:

- Migrate schedule and matrix reads and writes.
- Migrate worker claim logic with row-locking tests.
- Migrate worker heartbeat updates.
- Migrate cancellation convergence.
- Migrate run-token issue, validation, and revocation.
- Migrate agent registration and status APIs.
- Migrate runner-only endpoints.
- Add clock-skew, lease-expiry, race, load, and failure-injection tests.

Exit criteria:

- No run can be claimed twice.
- Stale workers are detected deterministically.
- Cancellation and finalization converge under retries.

## Wave 9 Epic: Browser Authentication And Administration

Goal: move browser-visible identity and administration last, after stateless
surfaces have proven stable.

Tracks:

- Login, logout, sessions, cookies, CSRF.
- Current user, menu, customer switching.
- Accounts, roles, profiles, password policy.
- Customer administration and legacy API-key lifecycle.
- Service accounts and credentials.
- MFA, SSO, OIDC, SCIM, workload identity, and break-glass controls.

Tickets:

- Build the temporary Laravel browser-auth introspection bridge.
- Migrate current-user and capability-resolution endpoints.
- Migrate login, logout, session expiry, and CSRF behavior.
- Migrate customer switching.
- Migrate accounts, roles, profiles, and password policy.
- Migrate customer administration.
- Migrate legacy API-key lifecycle.
- Migrate service accounts and scoped credentials.
- Migrate MFA and step-up authentication.
- Migrate SSO, OIDC callbacks, SCIM, workload identity, and break-glass controls.
- Remove the temporary auth bridge after compatibility passes.

Exit criteria:

- The unmodified Web client passes login, reload, logout, expiry, customer
  switch, CORS, and CSRF tests.
- Negative authorization and tenant-isolation suites pass for every role.
- Credential rotation and last-used timestamps remain correct.

## Wave 10 Epic: Schema Handover And Final Cutover

Goal: make Go the only API runtime and schema owner.

Tracks:

- Go migration baseline.
- Empty install and upgrade verification.
- Route ownership cutover.
- Laravel queue drain and write freeze.
- Docker default image switch.
- Rollback rehearsal and operations documentation.

Tickets:

- Freeze Laravel schema changes.
- Produce the reviewed Go baseline migration.
- Add a bridge command that marks a migrated Laravel schema as applied.
- Verify empty installs using Go migrations.
- Verify upgrades from the last Laravel-owned release.
- Move all remaining route ownership to Go in staging.
- Rehearse rollback to the last dual-runtime release.
- Switch Docker defaults to the Go API image.
- Update backup, recovery, release, and operations documentation.
- Enter a time-boxed Laravel read-only maintenance period before archival.

Exit criteria:

- No production route depends on Laravel.
- Compatibility, security, load, and recovery gates pass.
- Rollback has been rehearsed without requiring database restore.

## Recommended GitHub Issue Labels

- `wave-0` through `wave-10`
- `track-contract`
- `track-platform-catalog`
- `track-cli-read`
- `track-cli-write`
- `track-authoring`
- `track-results`
- `track-parallel`
- `track-auth`
- `track-schema`
- `security`
- `tenant-isolation`
- `compatibility`
- `docker`
- `web-smoke`
- `cli-smoke`
- `needs-differential-test`

## Next Issue Batch

Create these GitHub issues next:

1. Epic: Wave 0 - Baseline and governance.
2. Epic: Wave 2 - Contract and differential harness.
3. Epic: Wave 3 - Low-risk stateless reads.
4. Track: Platform catalog reads.
5. Ticket: Migrate location reads to Go.
6. Ticket: Migrate brand reads to Go.
7. Ticket: Migrate model reads to Go.
8. Ticket: Migrate OS and OS-version reads to Go.
9. Ticket: Migrate browser and browser-version reads to Go.
10. Ticket: Add Laravel-Go differential tests for platform catalog reads.
