# CLI Smoke Target Plan

This generated plan defines how Idelium CLI smoke tests choose Laravel
or Go for each remote execution route during the strangler migration.
The route owner recorded in the migration ownership matrix is
authoritative; unknown owners fail closed instead of falling back silently.

## Execution policy

- Consumer: `idelium-cli`
- Laravel base URL environment variable: `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL`
- Go base URL environment variable: `IDELIUM_CLI_SMOKE_GO_BASE_URL`
- Runtime credentials stay outside the generated plan and must be injected by CI or the local shell.
- Configuration reads validate remote CLI setup before a route owner changes.
- Result-reporting writes must use synthetic execution data and reversible fixture records.

## Summary

| Metric | Value |
| --- | --- |
| Routes | 13 |
| Owners | go: 3, laravel: 10 |
| Execution modes | configuration-read: 7, result-reporting-write: 6 |

## Routes

| Method | Path | Owner | Target env | Mode | Tenant | Aggregate |
| --- | --- | --- | --- | --- | --- | --- |
| `GET` | `/api/ideliumcl/step/{idStep}` | `go` | `IDELIUM_CLI_SMOKE_GO_BASE_URL` | `configuration-read` | no | `steps` |
| `GET` | `/api/ideliumcl/test/{idTest}` | `go` | `IDELIUM_CLI_SMOKE_GO_BASE_URL` | `configuration-read` | no | `tests` |
| `GET` | `/api/ideliumcl/testcycle/{idTestCycle}` | `go` | `IDELIUM_CLI_SMOKE_GO_BASE_URL` | `configuration-read` | no | `test-cycles` |
| `GET` | `/api/ideliumcl/environment/{idEnvironment}` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `configuration-read` | no | `environments` |
| `GET` | `/api/ideliumcl/environments/{idProject}` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `configuration-read` | no | `environments` |
| `GET` | `/api/ideliumcl/plugin/{idPlugin}` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `configuration-read` | no | `plugins` |
| `GET` | `/api/ideliumcl/plugins/{idProject}` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `configuration-read` | no | `plugins` |
| `POST` | `/api/ideliumcl/step` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `result-reporting-write` | no | `execution-results` |
| `PUT` | `/api/ideliumcl/step` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `result-reporting-write` | no | `execution-results` |
| `POST` | `/api/ideliumcl/test` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `result-reporting-write` | no | `execution-results` |
| `PUT` | `/api/ideliumcl/test` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `result-reporting-write` | no | `execution-results` |
| `POST` | `/api/ideliumcl/testcycle` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `result-reporting-write` | no | `execution-results` |
| `PUT` | `/api/ideliumcl/testcycle` | `laravel` | `IDELIUM_CLI_SMOKE_LARAVEL_BASE_URL` | `result-reporting-write` | no | `execution-results` |

## Compatibility and rollback

This plan does not move route traffic by itself. It lets CLI smoke tests
exercise the runtime assigned by the ownership matrix while Laravel remains
the fallback owner during coexistence. Rollback is a normal revert of the
generated plan or a route-owner change in the ownership matrix before smoke
execution.
