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
| Wave 1 | [#2](https://github.com/ifuschini/idelium-api-go/issues/2) | In progress | `e4e5def feat: bootstrap Go API foundation` |
| Wave 2 | [#3](https://github.com/ifuschini/idelium-api-go/issues/3) | Planned | Backlog materialized |
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


## Update policy

Update this cursor whenever a migration ticket is completed. Each completed
ticket must include verification evidence, a dedicated commit, and a GitHub
closure comment. Regenerate the mapping with:

```sh
python3 scripts/sync_migration_issues.py --repo ifuschini/idelium-api-go --apply
```
