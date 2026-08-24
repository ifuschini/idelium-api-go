# Migration Ownership Matrix

This generated matrix assigns every production-visible Laravel route to
one aggregate and one effective owner. An aggregate may have no mutations,
but an aggregate with mutations must have exactly one mutation owner.
Application-level dual writes are prohibited.

## Current summary

| Measure | Value |
| --- | ---: |
| Aggregates | 25 |
| Routes | 168 |
| Mutation routes | 98 |

## Aggregate ownership

| Aggregate | Mutation owner | Routes | Mutations | Tenant-scoped | State |
| --- | --- | ---: | ---: | ---: | --- |
| `access-control` | none | 3 | 0 | 3 | laravel-primary |
| `accounts` | laravel | 4 | 3 | 4 | laravel-primary |
| `agent-registry` | laravel | 3 | 2 | 2 | laravel-primary |
| `artifacts` | laravel | 7 | 4 | 7 | laravel-primary |
| `asset-versions` | laravel | 5 | 1 | 5 | laravel-primary |
| `audit-events` | none | 1 | 0 | 1 | laravel-primary |
| `browser-identity` | laravel | 8 | 6 | 7 | laravel-primary |
| `customers` | laravel | 6 | 4 | 6 | laravel-primary |
| `enterprise-identity` | laravel | 9 | 8 | 5 | laravel-primary |
| `environments` | laravel | 7 | 3 | 5 | laravel-primary |
| `execution-results` | laravel | 9 | 6 | 3 | laravel-primary |
| `grid-jobs` | laravel | 4 | 2 | 4 | laravel-primary |
| `integrations` | laravel | 7 | 5 | 7 | laravel-primary |
| `legacy-api-keys` | laravel | 2 | 1 | 2 | laravel-primary |
| `operations` | none | 4 | 0 | 0 | laravel-primary |
| `parallel-runs` | laravel | 23 | 17 | 9 | laravel-primary |
| `platform-catalog` | laravel | 27 | 17 | 27 | laravel-primary |
| `plugins` | laravel | 7 | 3 | 5 | laravel-primary |
| `projects` | laravel | 7 | 3 | 7 | laravel-primary |
| `result-exports` | laravel | 3 | 1 | 3 | laravel-primary |
| `service-accounts` | laravel | 3 | 2 | 3 | laravel-primary |
| `steps` | laravel | 7 | 4 | 6 | laravel-primary |
| `test-cycles` | laravel | 5 | 2 | 4 | laravel-primary |
| `test-launches` | laravel | 1 | 1 | 1 | laravel-primary |
| `tests` | laravel | 6 | 3 | 5 | laravel-primary |

## Ownership transition rule

A route changes to `go` only after its compatibility, authorization, tenant
isolation, side-effect, observability, and rollback evidence is approved.
All mutation routes in the same transaction-owning aggregate move together,
unless an ADR defines a smaller aggregate boundary. The gateway is the only
switch: neither runtime may replicate application writes into the other.

During rollback, route ownership returns to Laravel before the Go writer is
disabled. Shared schema changes must remain backward compatible, so rollback
does not require database restoration or reverse data replication.

## Route assignments

| Method | Path | Aggregate | Kind | Owner | Tenant | Wave |
| --- | --- | --- | --- | --- | --- | ---: |
| `GET|HEAD` | `/` | `operations` | read | laravel | no | 0 |
| `GET|HEAD` | `/api/admin/accounts` | `accounts` | read | laravel | yes | 9 |
| `POST` | `/api/admin/accounts` | `accounts` | mutation | laravel | yes | 9 |
| `DELETE` | `/api/admin/accounts/{idUser}` | `accounts` | mutation | laravel | yes | 9 |
| `PUT` | `/api/admin/accounts/{idUser}` | `accounts` | mutation | laravel | yes | 9 |
| `GET|HEAD` | `/api/admin/agents` | `agent-registry` | read | laravel | yes | 8 |
| `PUT` | `/api/admin/agents/{agentRegistration}/status` | `agent-registry` | mutation | laravel | yes | 8 |
| `GET|HEAD` | `/api/admin/apikey` | `legacy-api-keys` | read | laravel | yes | 9 |
| `PUT` | `/api/admin/apikey` | `legacy-api-keys` | mutation | laravel | yes | 9 |
| `GET|HEAD` | `/api/admin/costumers` | `customers` | read | laravel | yes | 9 |
| `POST` | `/api/admin/costumers` | `customers` | mutation | laravel | yes | 9 |
| `DELETE` | `/api/admin/costumers/{idCostumer}` | `customers` | mutation | laravel | yes | 9 |
| `PUT` | `/api/admin/costumers/{idCostumer}` | `customers` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/environments` | `environments` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/environments/{idProject}` | `environments` | read | laravel | yes | 6 |
| `DELETE` | `/api/admin/environments/{idProject}/{environment}` | `environments` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/environments/{idProject}/{environment}` | `environments` | read | laravel | yes | 6 |
| `PUT` | `/api/admin/environments/{idProject}/{environment}` | `environments` | mutation | laravel | yes | 6 |
| `POST` | `/api/admin/grid/bulk-jobs` | `grid-jobs` | mutation | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}` | `grid-jobs` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}/export` | `grid-jobs` | read | laravel | yes | 7 |
| `POST` | `/api/admin/grid/query-snapshots` | `grid-jobs` | mutation | laravel | yes | 7 |
| `PUT` | `/api/admin/identity/accounts/{user}/break-glass` | `enterprise-identity` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/identity/accounts/{user}/break-glass/test` | `enterprise-identity` | mutation | laravel | yes | 9 |
| `GET|HEAD` | `/api/admin/identity/providers` | `enterprise-identity` | read | laravel | yes | 9 |
| `POST` | `/api/admin/identity/providers` | `enterprise-identity` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/identity/providers/{identityProvider}/scim/users` | `enterprise-identity` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/importtest` | `tests` | mutation | laravel | yes | 6 |
| `POST` | `/api/admin/launchtest` | `test-launches` | mutation | laravel | yes | 8 |
| `GET|HEAD` | `/api/admin/platforms/brands` | `platform-catalog` | read | laravel | yes | 3 |
| `POST` | `/api/admin/platforms/brands` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/brands` | `platform-catalog` | mutation | laravel | yes | 6 |
| `POST` | `/api/admin/platforms/browsers` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/browsers` | `platform-catalog` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/platforms/browsers/{idOs}` | `platform-catalog` | read | laravel | yes | 3 |
| `POST` | `/api/admin/platforms/browserversions` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/browserversions` | `platform-catalog` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/platforms/browserversions/{idBrowser}` | `platform-catalog` | read | laravel | yes | 3 |
| `GET|HEAD` | `/api/admin/platforms/locations` | `platform-catalog` | read | laravel | yes | 3 |
| `POST` | `/api/admin/platforms/locations` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/locations` | `platform-catalog` | mutation | laravel | yes | 6 |
| `POST` | `/api/admin/platforms/manageplatforms` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/manageplatforms` | `platform-catalog` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/platforms/manageplatforms/{type}` | `platform-catalog` | read | laravel | yes | 3 |
| `DELETE` | `/api/admin/platforms/manageplatforms/{type}/{id}` | `platform-catalog` | mutation | laravel | yes | 6 |
| `POST` | `/api/admin/platforms/models` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/models` | `platform-catalog` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/platforms/models/{idBrand}` | `platform-catalog` | read | laravel | yes | 3 |
| `POST` | `/api/admin/platforms/os` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/os` | `platform-catalog` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/platforms/os/{idType}` | `platform-catalog` | read | laravel | yes | 3 |
| `POST` | `/api/admin/platforms/osversion` | `platform-catalog` | mutation | laravel | yes | 6 |
| `PUT` | `/api/admin/platforms/osversion` | `platform-catalog` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/platforms/osversion/{idOs}` | `platform-catalog` | read | laravel | yes | 3 |
| `GET|HEAD` | `/api/admin/platforms/status` | `platform-catalog` | read | laravel | yes | 3 |
| `GET|HEAD` | `/api/admin/platforms/types` | `platform-catalog` | read | go | yes | 3 |
| `POST` | `/api/admin/plugins` | `plugins` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/plugins/{idProject}` | `plugins` | read | laravel | yes | 6 |
| `DELETE` | `/api/admin/plugins/{idProject}/{plugin}` | `plugins` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/plugins/{idProject}/{plugin}` | `plugins` | read | laravel | yes | 6 |
| `PUT` | `/api/admin/plugins/{idProject}/{step}` | `plugins` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/profile` | `browser-identity` | read | laravel | yes | 9 |
| `PUT` | `/api/admin/profile` | `browser-identity` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/profile/mfa/confirm` | `browser-identity` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/profile/mfa/enroll` | `browser-identity` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/profile/mfa/step-up` | `browser-identity` | mutation | laravel | yes | 9 |
| `GET|HEAD` | `/api/admin/projects` | `projects` | read | laravel | yes | 6 |
| `POST` | `/api/admin/projects` | `projects` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/projects/create` | `projects` | read | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-impact/{assetType}/{assetId}` | `asset-versions` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetType}/{assetId}` | `asset-versions` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}` | `asset-versions` | read | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}/review-events` | `asset-versions` | mutation | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{fromVersion}/diff/{toVersion}` | `asset-versions` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integration-deliveries` | `integrations` | read | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/integration-deliveries/{integrationDelivery}/replay` | `integrations` | mutation | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integrations` | `integrations` | read | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/integrations` | `integrations` | mutation | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/rotate-secret` | `integrations` | mutation | laravel | yes | 7 |
| `PUT` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/status` | `integrations` | mutation | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/test` | `integrations` | mutation | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs` | `parallel-runs` | read | laravel | yes | 8 |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs` | `parallel-runs` | mutation | laravel | yes | 8 |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/matrix` | `parallel-runs` | mutation | laravel | yes | 8 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}` | `parallel-runs` | read | laravel | yes | 8 |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `parallel-runs` | mutation | laravel | yes | 8 |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `parallel-runs` | mutation | laravel | yes | 8 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/results` | `parallel-runs` | read | laravel | yes | 8 |
| `PUT` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `parallel-runs` | mutation | laravel | yes | 8 |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `parallel-runs` | mutation | laravel | yes | 8 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | `artifacts` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | `artifacts` | read | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | `artifacts` | mutation | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | `artifacts` | mutation | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/impact` | `artifacts` | read | laravel | yes | 7 |
| `PUT` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | `artifacts` | mutation | laravel | yes | 7 |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | `artifacts` | mutation | laravel | yes | 7 |
| `DELETE` | `/api/admin/projects/{project}` | `projects` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/projects/{project}` | `projects` | read | laravel | yes | 6 |
| `PUT|PATCH` | `/api/admin/projects/{project}` | `projects` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/projects/{project}/edit` | `projects` | read | laravel | yes | 6 |
| `POST` | `/api/admin/result-exports` | `result-exports` | mutation | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}` | `result-exports` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}/download` | `result-exports` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/admin/roles` | `access-control` | read | laravel | yes | 9 |
| `GET|HEAD` | `/api/admin/service-accounts` | `service-accounts` | read | laravel | yes | 9 |
| `POST` | `/api/admin/service-accounts` | `service-accounts` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/service-accounts/{serviceAccount}/revoke` | `service-accounts` | mutation | laravel | yes | 9 |
| `POST` | `/api/admin/steps` | `steps` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/steps/{idProject}` | `steps` | read | laravel | yes | 6 |
| `POST` | `/api/admin/steps/{idProject}/updateorder` | `steps` | mutation | laravel | yes | 6 |
| `DELETE` | `/api/admin/steps/{idProject}/{environment}` | `steps` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/steps/{idProject}/{step}` | `steps` | read | laravel | yes | 6 |
| `PUT` | `/api/admin/steps/{idProject}/{step}` | `steps` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/stepsperfomed/{idTestPerformed}` | `execution-results` | read | laravel | yes | 7 |
| `POST` | `/api/admin/testcycles` | `test-cycles` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}` | `test-cycles` | read | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}/{testcycle}` | `test-cycles` | read | laravel | yes | 6 |
| `PUT` | `/api/admin/testcycles/{idProject}/{testcycle}` | `test-cycles` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/testcyclesperfomed/{idTestCyclePerformed}` | `execution-results` | read | laravel | yes | 7 |
| `POST` | `/api/admin/tests` | `tests` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/tests/{idProject}` | `tests` | read | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/tests/{idProject}/{test}` | `tests` | read | laravel | yes | 6 |
| `PUT` | `/api/admin/tests/{idProject}/{test}` | `tests` | mutation | laravel | yes | 6 |
| `GET|HEAD` | `/api/admin/testsperfomed/{idTestPerformed}` | `execution-results` | read | laravel | yes | 7 |
| `GET|HEAD` | `/api/audit-events` | `audit-events` | read | laravel | yes | 9 |
| `GET|HEAD` | `/api/clear` | `operations` | read | laravel | no | 0 |
| `GET|HEAD` | `/api/csrf-cookie` | `operations` | read | laravel | no | 0 |
| `POST` | `/api/ideliumcl/agents/register` | `agent-registry` | mutation | laravel | no | 8 |
| `GET|HEAD` | `/api/ideliumcl/environment/{idEnvironment}` | `environments` | read | laravel | no | 4 |
| `GET|HEAD` | `/api/ideliumcl/environments/{idProject}` | `environments` | read | laravel | no | 4 |
| `GET|HEAD` | `/api/ideliumcl/plugin/{idPlugin}` | `plugins` | read | laravel | no | 4 |
| `GET|HEAD` | `/api/ideliumcl/plugins/{idProject}` | `plugins` | read | laravel | no | 4 |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `parallel-runs` | read | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/matrix` | `parallel-runs` | mutation | laravel | no | 8 |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}` | `parallel-runs` | read | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `parallel-runs` | mutation | laravel | no | 8 |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/results` | `parallel-runs` | read | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens/{tokenId}/revoke` | `parallel-runs` | mutation | laravel | no | 8 |
| `PUT` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/ideliumcl/step` | `execution-results` | mutation | laravel | no | 5 |
| `PUT` | `/api/ideliumcl/step` | `execution-results` | mutation | laravel | no | 5 |
| `GET|HEAD` | `/api/ideliumcl/step/{idStep}` | `steps` | read | laravel | no | 4 |
| `POST` | `/api/ideliumcl/test` | `execution-results` | mutation | laravel | no | 5 |
| `PUT` | `/api/ideliumcl/test` | `execution-results` | mutation | laravel | no | 5 |
| `GET|HEAD` | `/api/ideliumcl/test/{idTest}` | `tests` | read | laravel | no | 4 |
| `POST` | `/api/ideliumcl/testcycle` | `execution-results` | mutation | laravel | no | 5 |
| `PUT` | `/api/ideliumcl/testcycle` | `execution-results` | mutation | laravel | no | 5 |
| `GET|HEAD` | `/api/ideliumcl/testcycle/{idTestCycle}` | `test-cycles` | read | laravel | no | 4 |
| `POST` | `/api/ideliumrunner/claim` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/ideliumrunner/heartbeat` | `parallel-runs` | mutation | laravel | no | 8 |
| `PUT` | `/api/ideliumrunner/worker` | `parallel-runs` | mutation | laravel | no | 8 |
| `POST` | `/api/login` | `browser-identity` | mutation | laravel | no | 9 |
| `POST` | `/api/logout` | `browser-identity` | mutation | laravel | yes | 9 |
| `GET|HEAD` | `/api/me/capabilities` | `access-control` | read | laravel | yes | 9 |
| `GET|HEAD` | `/api/menu/header` | `customers` | read | laravel | yes | 9 |
| `PUT` | `/api/menu/header/{idCostumer}` | `customers` | mutation | laravel | yes | 9 |
| `GET|HEAD` | `/api/menu/sidebar` | `access-control` | read | laravel | yes | 9 |
| `POST` | `/api/oidc/token-exchange` | `enterprise-identity` | mutation | laravel | no | 9 |
| `GET|HEAD` | `/api/sanctum/csrf-cookie` | `operations` | read | laravel | no | 9 |
| `POST` | `/api/sso/{identityProvider}/oidc/callback` | `enterprise-identity` | mutation | laravel | no | 9 |
| `POST` | `/api/sso/{identityProvider}/saml/callback` | `enterprise-identity` | mutation | laravel | no | 9 |
| `POST` | `/api/sso/{identityProvider}/start` | `enterprise-identity` | mutation | laravel | no | 9 |
| `GET|HEAD` | `/api/user` | `browser-identity` | read | laravel | yes | 9 |

## Deployment and rollback

This baseline is governance-only: it moves no traffic, performs no writes,
and changes no database schema. Laravel remains the effective and fallback
owner for every recorded route. Deployment publishes the matrix; rollback is
a Git revert. Differential HTTP testing is not applicable until a route is
implemented in both runtimes.
