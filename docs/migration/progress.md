# Idelium API Go Migration Progress

This file records the current operational cursor for the Laravel-to-Go migration.
The detailed strategy remains in [`MIGRATION_PLAN.md`](../../MIGRATION_PLAN.md),
while [`epics.md`](epics.md) is the versioned backlog source.

## GitHub backlog

- Repository: https://github.com/ifuschini/idelium-api-go
- Wave epics: 11
- Domain tracks: 60
- Implementation tickets: 107
- Machine-readable mapping: [`github-issues.json`](github-issues.json)

## Current cursor

| Wave | GitHub epic | Status | Evidence |
| --- | --- | --- | --- |
| Wave 0 | [#1](https://github.com/ifuschini/idelium-api-go/issues/1) | Complete | Tracks [#12](https://github.com/ifuschini/idelium-api-go/issues/12), [#13](https://github.com/ifuschini/idelium-api-go/issues/13), [#14](https://github.com/ifuschini/idelium-api-go/issues/14), and [#15](https://github.com/ifuschini/idelium-api-go/issues/15) completed |
| Wave 1 | [#2](https://github.com/ifuschini/idelium-api-go/issues/2) | Complete | Tracks [#16](https://github.com/ifuschini/idelium-api-go/issues/16), [#17](https://github.com/ifuschini/idelium-api-go/issues/17), [#18](https://github.com/ifuschini/idelium-api-go/issues/18), [#19](https://github.com/ifuschini/idelium-api-go/issues/19), and [#20](https://github.com/ifuschini/idelium-api-go/issues/20) completed |
| Wave 2 | [#3](https://github.com/ifuschini/idelium-api-go/issues/3) | In progress | Tickets [#89](https://github.com/ifuschini/idelium-api-go/issues/89) and [#90](https://github.com/ifuschini/idelium-api-go/issues/90) completed |
| Wave 3 | [#4](https://github.com/ifuschini/idelium-api-go/issues/4) | In progress | `d4b7f22 feat: add read-only platform catalogs` |
| Wave 4 | [#5](https://github.com/ifuschini/idelium-api-go/issues/5) | Planned | Backlog materialized |
| Wave 5 | [#6](https://github.com/ifuschini/idelium-api-go/issues/6) | Planned | Backlog materialized |
| Wave 6 | [#7](https://github.com/ifuschini/idelium-api-go/issues/7) | Planned | Backlog materialized |
| Wave 7 | [#8](https://github.com/ifuschini/idelium-api-go/issues/8) | Planned | Backlog materialized |
| Wave 8 | [#9](https://github.com/ifuschini/idelium-api-go/issues/9) | Planned | Backlog materialized |
| Wave 9 | [#10](https://github.com/ifuschini/idelium-api-go/issues/10) | Planned | Backlog materialized |
| Wave 10 | [#11](https://github.com/ifuschini/idelium-api-go/issues/11) | Planned | Backlog materialized |

## Completed tickets

| Ticket | Result | Verification |
| --- | --- | --- |
| [#72](https://github.com/ifuschini/idelium-api-go/issues/72) | Exported and classified all 171 routes registered by Laravel. | Generated JSON and Markdown inventories; exporter unit and integrity tests. |
| [#73](https://github.com/ifuschini/idelium-api-go/issues/73) | Mapped Web, CLI, runner, Docker, and wiki consumers to the Laravel inventory. | Generated route-level map, unresolved-reference register, and integrity tests. |
| [#74](https://github.com/ifuschini/idelium-api-go/issues/74) | Classified all Laravel routes into one of five canonical migration trust paths. | Generated classifications, exhaustive category assertions, and safe handling for unknown modes. |
| [#75](https://github.com/ifuschini/idelium-api-go/issues/75) | Established one compatibility-contract record for every production-visible Laravel route. | Generated 168 records, three explicit development-only exclusions, wave routing, and contract-gate tests. |
| [#76](https://github.com/ifuschini/idelium-api-go/issues/76) | Defined the initial sanitized golden fixture format and capture policy. | Machine-readable schema, safe example, bounded validator, and regression tests for credentials, tenant data, and payload limits. |
| [#77](https://github.com/ifuschini/idelium-api-go/issues/77) | Assigned every production-visible route to an aggregate and effective migration owner. | Generated 168 route assignments across 25 aggregates; tests reject implicit ownership and split mutation ownership. |
| [#78](https://github.com/ifuschini/idelium-api-go/issues/78) | Accepted the route-level strangler migration model and established the ADR lifecycle. | ADR template, indexed decision, and structural tests for compatibility, security, ownership, and rollback sections. |
| [#79](https://github.com/ifuschini/idelium-api-go/issues/79) | Defined release, approval, progressive route-switch, stop, and rollback gates. | Machine-readable policy and negative tests for dual writes, tenant incidents, incomplete approvals, and unsafe rollback ownership. |
| [#80](https://github.com/ifuschini/idelium-api-go/issues/80) | Completed the bounded API process lifecycle with build identity and graceful shutdown. | Testable server lifecycle, configured timeout assertions, cancellation regression test, and documented deployment and rollback behavior. |
| [#81](https://github.com/ifuschini/idelium-api-go/issues/81) | Added fail-closed worker and migration process skeletons for scheduled future owning domains. | Shared cancellable process lifecycle, safe build identity, ownership and handler validation, and activation gates that prevent premature work or migrations. |
| [#82](https://github.com/ifuschini/idelium-api-go/issues/82) | Completed compatible database configuration, secret-file precedence, validation, and redacted failures. | Unit tests cover precedence, unreadable paths, missing secrets, invalid bounds, and safe database failure classifications; MySQL integration verifies connectivity. |
| [#83](https://github.com/ifuschini/idelium-api-go/issues/83) | Stabilized non-cacheable liveness and bounded MySQL readiness contracts. | OpenAPI response headers, dependency-free liveness tests, deadline and redaction tests, and an HTTP readiness integration test against pinned MariaDB. |
| [#84](https://github.com/ifuschini/idelium-api-go/issues/84) | Completed the common HTTP safety and observability middleware contract. | Validated correlation propagation, redacted structured access logs, safe panic recovery, response security headers, OpenAPI response headers, and regression tests. |
| [#85](https://github.com/ifuschini/idelium-api-go/issues/85) | Completed the isolated MySQL readiness and integration-test infrastructure. | Pinned ephemeral MariaDB, real readiness HTTP coverage, schema isolation assertion, redacted authentication failure test, and documented future migration and tenant-test gates. |
| [#86](https://github.com/ifuschini/idelium-api-go/issues/86) | Locked the repository foundation and its engineering contract. | Automated checks preserve Apache 2.0 licensing, repository directives, README requirements, Make targets, and the pinned reproducible non-root Dockerfile. |
| [#87](https://github.com/ifuschini/idelium-api-go/issues/87) | Split CI into bounded quality, MySQL integration, and container-image gates. | Pinned actions and runtimes, contract and race coverage, isolated MariaDB tests, non-root image inspection, and workflow regression tests. |
| [#88](https://github.com/ifuschini/idelium-api-go/issues/88) | Added auditable SBOM and fail-closed vulnerability scanning gates. | Pinned govulncheck, Syft CycloneDX artifact generation, Trivy fixed high/critical image policy, immutable tool references, and workflow regression tests. |
| [#89](https://github.com/ifuschini/idelium-api-go/issues/89) | Expanded OpenAPI coverage to every production-visible Laravel route. | Generated compatibility contracts preserve Laravel route, controller, owner, trust path, authentication, tenant context, and consumer metadata; contract tests enforce 168/168 documented routes and synchronization with the committed Laravel inventory. |
| [#90](https://github.com/ifuschini/idelium-api-go/issues/90) | Added generated Go server contracts and drift checks from OpenAPI. | Generated `ServerInterface` and operation metadata for 171 OpenAPI operations; `make openapi-check` and contract tests fail when generated Go contracts drift from `api/openapi.yaml`. |


## Update policy

Update this cursor whenever a migration ticket is completed. Each completed
ticket must include verification evidence, a dedicated commit, and a GitHub
closure comment. Regenerate the mapping with:

```sh
python3 scripts/sync_migration_issues.py --repo ifuschini/idelium-api-go --apply
```
