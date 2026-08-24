# Web Smoke Target Plan

This generated plan defines how Idelium Web smoke tests choose Laravel
or Go for each route consumed by the Web console during the strangler
migration. The route owner recorded in the migration ownership matrix is
authoritative; unknown owners fail closed instead of falling back silently.

## Execution policy

- Consumer: `idelium-web`
- Laravel base URL environment variable: `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL`
- Go base URL environment variable: `IDELIUM_WEB_SMOKE_GO_BASE_URL`
- Smoke plans must not contain credentials, cookies, CSRF tokens, session IDs, or payload secrets.
- Browser-session routes require a synthetic test session created outside this generated plan.
- Mutation routes must use isolated synthetic tenants and reversible fixture data.

## Summary

| Metric | Value |
| --- | --- |
| Routes | 91 |
| Owners | laravel: 91 |
| Execution modes | safe-read: 38, synthetic-mutation: 53 |

## Routes

| Method | Path | Owner | Target env | Mode | Tenant | Aggregate |
| --- | --- | --- | --- | --- | --- | --- |
| `GET` | `/api/admin/accounts` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `accounts` |
| `POST` | `/api/admin/accounts` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `accounts` |
| `DELETE` | `/api/admin/accounts/{idUser}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `accounts` |
| `PUT` | `/api/admin/accounts/{idUser}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `accounts` |
| `GET` | `/api/admin/apikey` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `legacy-api-keys` |
| `PUT` | `/api/admin/apikey` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `legacy-api-keys` |
| `GET` | `/api/admin/costumers` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `customers` |
| `POST` | `/api/admin/costumers` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `customers` |
| `DELETE` | `/api/admin/costumers/{idCostumer}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `customers` |
| `PUT` | `/api/admin/costumers/{idCostumer}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `customers` |
| `POST` | `/api/admin/environments` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `environments` |
| `GET` | `/api/admin/environments/{idProject}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `environments` |
| `DELETE` | `/api/admin/environments/{idProject}/{environment}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `environments` |
| `GET` | `/api/admin/environments/{idProject}/{environment}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `environments` |
| `PUT` | `/api/admin/environments/{idProject}/{environment}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `environments` |
| `POST` | `/api/admin/importtest` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `tests` |
| `POST` | `/api/admin/launchtest` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `test-launches` |
| `GET` | `/api/admin/platforms/brands` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/brands` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/brands` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/browsers` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/browsers` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/browsers/{idOs}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/browserversions` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/browserversions` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/browserversions/{idBrowser}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/locations` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/locations` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/locations` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/manageplatforms` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/manageplatforms` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/manageplatforms/{type}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `DELETE` | `/api/admin/platforms/manageplatforms/{type}/{id}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/models` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/models` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/models/{idBrand}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/os` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/os` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/os/{idType}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `POST` | `/api/admin/platforms/osversion` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `PUT` | `/api/admin/platforms/osversion` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/osversion/{idOs}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/status` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `GET` | `/api/admin/platforms/types` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `platform-catalog` |
| `POST` | `/api/admin/plugins` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `plugins` |
| `GET` | `/api/admin/plugins/{idProject}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `plugins` |
| `DELETE` | `/api/admin/plugins/{idProject}/{plugin}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `plugins` |
| `GET` | `/api/admin/plugins/{idProject}/{plugin}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `plugins` |
| `PUT` | `/api/admin/plugins/{idProject}/{step}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `plugins` |
| `GET` | `/api/admin/profile` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `browser-identity` |
| `PUT` | `/api/admin/profile` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `browser-identity` |
| `GET` | `/api/admin/projects` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `projects` |
| `POST` | `/api/admin/projects` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `projects` |
| `GET` | `/api/admin/projects/create` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `projects` |
| `GET` | `/api/admin/projects/{idProject}/parallel-runs` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `parallel-runs` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `parallel-runs` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/matrix` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `parallel-runs` |
| `GET` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `parallel-runs` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `parallel-runs` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `parallel-runs` |
| `GET` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/results` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `parallel-runs` |
| `PUT` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `parallel-runs` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `parallel-runs` |
| `DELETE` | `/api/admin/projects/{project}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `projects` |
| `GET` | `/api/admin/projects/{project}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `projects` |
| `PUT|PATCH` | `/api/admin/projects/{project}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `projects` |
| `GET` | `/api/admin/projects/{project}/edit` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `projects` |
| `GET` | `/api/admin/roles` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `access-control` |
| `POST` | `/api/admin/steps` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `steps` |
| `GET` | `/api/admin/steps/{idProject}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `steps` |
| `POST` | `/api/admin/steps/{idProject}/updateorder` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `steps` |
| `DELETE` | `/api/admin/steps/{idProject}/{environment}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `steps` |
| `GET` | `/api/admin/steps/{idProject}/{step}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `steps` |
| `PUT` | `/api/admin/steps/{idProject}/{step}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `steps` |
| `GET` | `/api/admin/stepsperfomed/{idTestPerformed}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `execution-results` |
| `POST` | `/api/admin/testcycles` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `test-cycles` |
| `GET` | `/api/admin/testcycles/{idProject}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `test-cycles` |
| `GET` | `/api/admin/testcycles/{idProject}/{testcycle}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `test-cycles` |
| `PUT` | `/api/admin/testcycles/{idProject}/{testcycle}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `test-cycles` |
| `GET` | `/api/admin/testcyclesperfomed/{idTestCyclePerformed}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `execution-results` |
| `POST` | `/api/admin/tests` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `tests` |
| `GET` | `/api/admin/tests/{idProject}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `tests` |
| `GET` | `/api/admin/tests/{idProject}/{test}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `tests` |
| `PUT` | `/api/admin/tests/{idProject}/{test}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `tests` |
| `GET` | `/api/admin/testsperfomed/{idTestPerformed}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `execution-results` |
| `POST` | `/api/login` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | no | `browser-identity` |
| `POST` | `/api/logout` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `browser-identity` |
| `GET` | `/api/menu/header` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `customers` |
| `PUT` | `/api/menu/header/{idCostumer}` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `synthetic-mutation` | yes | `customers` |
| `GET` | `/api/menu/sidebar` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | yes | `access-control` |
| `GET` | `/api/sanctum/csrf-cookie` | `laravel` | `IDELIUM_WEB_SMOKE_LARAVEL_BASE_URL` | `safe-read` | no | `operations` |

## Compatibility and rollback

This plan does not move traffic by itself. It gives Web smoke tests a
deterministic target for each route while Laravel remains the fallback owner
during coexistence. Rollback is a normal revert of the generated plan or a
route-owner change in the ownership matrix before smoke execution.
