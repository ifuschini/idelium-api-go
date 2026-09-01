# Compatibility Contract Backlog

This generated backlog creates one compatibility record for every Laravel route
that is reachable outside development-only tooling. It is the contract gate for
moving route ownership to Go; a route cannot move while required evidence remains
pending.

## Summary

- Public Laravel route records: **168**
- Excluded development-only routes: **3**
- OpenAPI-documented Laravel operations: **168**
- Operations awaiting OpenAPI contracts: **0**

| Priority | Records |
| --- | ---: |
| `critical` | 2 |
| `high` | 121 |
| `normal` | 45 |

| Migration wave | Records |
| --- | ---: |
| Wave 0 | 3 |
| Wave 3 | 10 |
| Wave 4 | 7 |
| Wave 5 | 6 |
| Wave 6 | 49 |
| Wave 7 | 29 |
| Wave 8 | 27 |
| Wave 9 | 37 |

## Contract gates

Each record requires request validation, response/status behavior, authorization
and tenant isolation, side effects and idempotency, redaction and audit behavior,
a sanitized Laravel fixture, a Laravel-Go differential test, applicable consumer
smoke coverage, and explicit rollout/rollback evidence.

All production-visible Laravel routes are present in the Go OpenAPI document.
Compatibility placeholder operations preserve ownership and consumer metadata;
fixtures and differential tests intentionally remain pending until the Wave 2
harness is available.

## Route backlog

| Priority | Wave | Method | Path | Trust path | Consumers | OpenAPI | Owner |
| --- | ---: | --- | --- | --- | --- | --- | --- |
| `high` | 0 | `GET|HEAD` | `/` | `public-operational` | — | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/admin/accounts` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `POST` | `/api/admin/accounts` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `DELETE` | `/api/admin/accounts/{idUser}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `PUT` | `/api/admin/accounts/{idUser}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 8 | `GET|HEAD` | `/api/admin/agents` | `browser-session` | — | `documented` | `go-application` |
| `normal` | 8 | `PUT` | `/api/admin/agents/{agentRegistration}/status` | `browser-session` | — | `documented` | `go-application` |
| `high` | 9 | `GET|HEAD` | `/api/admin/apikey` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `PUT` | `/api/admin/apikey` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/admin/costumers` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `POST` | `/api/admin/costumers` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `DELETE` | `/api/admin/costumers/{idCostumer}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `PUT` | `/api/admin/costumers/{idCostumer}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/environments` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/environments/{idProject}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `DELETE` | `/api/admin/environments/{idProject}/{environment}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/environments/{idProject}/{environment}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/environments/{idProject}/{environment}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/grid/bulk-jobs` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}/export` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/grid/query-snapshots` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `PUT` | `/api/admin/identity/accounts/{user}/break-glass` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/identity/accounts/{user}/break-glass/test` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `GET|HEAD` | `/api/admin/identity/providers` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/identity/providers` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/identity/providers/{identityProvider}/scim/users` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/importtest` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/admin/launchtest` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/brands` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/brands` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/brands` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/browsers` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/browsers` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/browsers/{idOs}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/browserversions` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/browserversions` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/browserversions/{idBrowser}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/locations` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/locations` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/locations` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/manageplatforms` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/manageplatforms` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/manageplatforms/{type}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `DELETE` | `/api/admin/platforms/manageplatforms/{type}/{id}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/models` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/models` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/models/{idBrand}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/os` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/os` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/os/{idType}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/platforms/osversion` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/platforms/osversion` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/osversion/{idOs}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/status` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 3 | `GET|HEAD` | `/api/admin/platforms/types` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/plugins` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/plugins/{idProject}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `DELETE` | `/api/admin/plugins/{idProject}/{plugin}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/plugins/{idProject}/{plugin}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/plugins/{idProject}/{step}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/admin/profile` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `PUT` | `/api/admin/profile` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/profile/mfa/confirm` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/profile/mfa/enroll` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/profile/mfa/step-up` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/projects` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/projects` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/projects/create` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/asset-impact/{assetType}/{assetId}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetType}/{assetId}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}/review-events` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{fromVersion}/diff/{toVersion}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/integration-deliveries` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/integration-deliveries/{integrationDelivery}/replay` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/integrations` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/integrations` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/rotate-secret` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `PUT` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/status` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/test` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 8 | `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/admin/projects/{idProject}/parallel-runs` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/admin/projects/{idProject}/parallel-runs/matrix` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/results` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `PUT` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/impact` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `PUT` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 6 | `DELETE` | `/api/admin/projects/{project}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/projects/{project}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT|PATCH` | `/api/admin/projects/{project}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/projects/{project}/edit` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 7 | `POST` | `/api/admin/result-exports` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/result-exports/{resultExport}` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 7 | `GET|HEAD` | `/api/admin/result-exports/{resultExport}/download` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/admin/roles` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 9 | `GET|HEAD` | `/api/admin/service-accounts` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/service-accounts` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/admin/service-accounts/{serviceAccount}/revoke` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/steps` | `browser-session` | `idelium-web` | `documented` | `go-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/steps/{idProject}` | `browser-session` | `idelium-web` | `documented` | `go-application` |
| `high` | 6 | `POST` | `/api/admin/steps/{idProject}/updateorder` | `browser-session` | `idelium-web` | `documented` | `go-application` |
| `high` | 6 | `DELETE` | `/api/admin/steps/{idProject}/{environment}` | `browser-session` | `idelium-web` | `documented` | `go-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/steps/{idProject}/{step}` | `browser-session` | `idelium-web` | `documented` | `go-application` |
| `high` | 6 | `PUT` | `/api/admin/steps/{idProject}/{step}` | `browser-session` | `idelium-web` | `documented` | `go-application` |
| `high` | 7 | `GET|HEAD` | `/api/admin/stepsperfomed/{idTestPerformed}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/testcycles` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/testcycles/{idProject}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/testcycles/{idProject}/{testcycle}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/testcycles/{idProject}/{testcycle}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 7 | `GET|HEAD` | `/api/admin/testcyclesperfomed/{idTestCyclePerformed}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `POST` | `/api/admin/tests` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/tests/{idProject}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `GET|HEAD` | `/api/admin/tests/{idProject}/{test}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 6 | `PUT` | `/api/admin/tests/{idProject}/{test}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 7 | `GET|HEAD` | `/api/admin/testsperfomed/{idTestPerformed}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 9 | `GET|HEAD` | `/api/audit-events` | `browser-session` | — | `documented` | `laravel-application` |
| `critical` | 0 | `GET|HEAD` | `/api/clear` | `public-operational` | — | `documented` | `laravel-application` |
| `high` | 0 | `GET|HEAD` | `/api/csrf-cookie` | `public-operational` | — | `documented` | `laravel-framework` |
| `high` | 8 | `POST` | `/api/ideliumcl/agents/register` | `api-key` | — | `documented` | `go-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/environment/{idEnvironment}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/environments/{idProject}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/plugin/{idPlugin}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/plugins/{idProject}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 8 | `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/matrix` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/results` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens/{tokenId}/revoke` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `PUT` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `api-key` | — | `documented` | `laravel-application` |
| `high` | 5 | `POST` | `/api/ideliumcl/step` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 5 | `PUT` | `/api/ideliumcl/step` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/step/{idStep}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 5 | `POST` | `/api/ideliumcl/test` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 5 | `PUT` | `/api/ideliumcl/test` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/test/{idTest}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 5 | `POST` | `/api/ideliumcl/testcycle` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 5 | `PUT` | `/api/ideliumcl/testcycle` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 4 | `GET|HEAD` | `/api/ideliumcl/testcycle/{idTestCycle}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | `documented` | `laravel-application` |
| `high` | 8 | `POST` | `/api/ideliumrunner/claim` | `run-token` | `idelium-runner` | `documented` | `go-application` |
| `high` | 8 | `POST` | `/api/ideliumrunner/heartbeat` | `run-token` | `idelium-runner` | `documented` | `go-application` |
| `high` | 8 | `PUT` | `/api/ideliumrunner/worker` | `run-token` | `idelium-runner` | `documented` | `go-application` |
| `high` | 9 | `POST` | `/api/login` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `POST` | `/api/logout` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `normal` | 9 | `GET|HEAD` | `/api/me/capabilities` | `browser-session` | — | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/menu/header` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `PUT` | `/api/menu/header/{idCostumer}` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/menu/sidebar` | `browser-session` | `idelium-web` | `documented` | `laravel-application` |
| `critical` | 9 | `POST` | `/api/oidc/token-exchange` | `internal-service` | — | `documented` | `laravel-application` |
| `high` | 9 | `GET|HEAD` | `/api/sanctum/csrf-cookie` | `browser-session` | `idelium-web`, `idelium-docker` | `documented` | `laravel-framework` |
| `normal` | 9 | `POST` | `/api/sso/{identityProvider}/oidc/callback` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/sso/{identityProvider}/saml/callback` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `POST` | `/api/sso/{identityProvider}/start` | `browser-session` | — | `documented` | `laravel-application` |
| `normal` | 9 | `GET|HEAD` | `/api/user` | `browser-session` | — | `documented` | `laravel-application` |

## Explicit exclusions

| Method | Path | Decision |
| --- | --- | --- |
| `POST` | `/_ignition/execute-solution` | Development-only Ignition route; prohibit it in production rather than migrate it. |
| `GET|HEAD` | `/_ignition/health-check` | Development-only Ignition route; prohibit it in production rather than migrate it. |
| `POST` | `/_ignition/update-config` | Development-only Ignition route; prohibit it in production rather than migrate it. |

## Deployment and rollback

This backlog changes no runtime behavior, traffic ownership, or database schema.
Rollback is a Git revert. Subsequent route migrations must update the relevant
record atomically with their contract, fixture, test, ownership, and rollback
evidence.

Regenerate the backlog whenever route, consumer, or OpenAPI evidence changes:

```sh
python3 scripts/build_compatibility_backlog.py \
  --inventory docs/contracts/laravel-routes.json \
  --consumer-map docs/contracts/consumer-route-map.json \
  --openapi api/openapi.yaml \
  --output-json docs/contracts/compatibility-backlog.json \
  --output-markdown docs/contracts/compatibility-backlog.md
```
