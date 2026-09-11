# Staging Route Cutover Manifest

This generated manifest is the staging checklist for moving route
ownership from Laravel to Go. It deliberately keeps production disabled
until every route is either implemented in Go or exposed as an explicit
Go fail-closed gate. Application-level dual writes remain prohibited.

## Status

| Field | Value |
| --- | --- |
| Cutover status | `ready` |
| Production enabled | `false` |
| Route count | 168 |
| Go-owned routes | 165 |
| Go fail-closed routes | 1 |
| Laravel blocker routes | 0 |
| Gateway Go routes | 164 |

## Staging policy

- Staging may route `ready` entries to Go.
- Staging may route `gated` entries to Go only when the expected
  fail-closed diagnostic is acceptable for that rehearsal.
- `blocked` entries stay on Laravel until a Go implementation or
  fail-closed gate is merged.
- Production cutover remains disabled until there are zero blockers.
- Dual writes are not allowed.

## Blocker summary by aggregate

| Aggregate | Blocked routes |
| --- | ---: |
| none | 0 |

## Route decisions

| Method | Path | Aggregate | State | Staging owner | Action |
| --- | --- | --- | --- | --- | --- |
| `GET|HEAD` | `/` | `operations` | `laravel-operational` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/accounts` | `accounts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/accounts` | `accounts` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/accounts/{idUser}` | `accounts` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/accounts/{idUser}` | `accounts` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/agents` | `agent-registry` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/agents/{agentRegistration}/status` | `agent-registry` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/apikey` | `legacy-api-keys` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/apikey` | `legacy-api-keys` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/costumers` | `customers` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/costumers` | `customers` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/costumers/{idCostumer}` | `customers` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/costumers/{idCostumer}` | `customers` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/environments` | `environments` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/environments/{idProject}` | `environments` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/environments/{idProject}/{environment}` | `environments` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/environments/{idProject}/{environment}` | `environments` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/environments/{idProject}/{environment}` | `environments` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/grid/bulk-jobs` | `grid-jobs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}` | `grid-jobs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}/export` | `grid-jobs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/grid/query-snapshots` | `grid-jobs` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/identity/accounts/{user}/break-glass` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/identity/accounts/{user}/break-glass/test` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/identity/providers` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/identity/providers` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/identity/providers/{identityProvider}/scim/users` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/importtest` | `tests` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/launchtest` | `test-launches` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `GET|HEAD` | `/api/admin/platforms/brands` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/brands` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/brands` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/browsers` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/browsers` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/browsers/{idOs}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/browserversions` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/browserversions` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/browserversions/{idBrowser}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/locations` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/locations` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/locations` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/manageplatforms` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/manageplatforms` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/manageplatforms/{type}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/platforms/manageplatforms/{type}/{id}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/models` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/models` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/models/{idBrand}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/os` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/os` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/os/{idType}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/osversion` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/platforms/osversion` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/osversion/{idOs}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/status` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/types` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/plugins` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/plugins/{idProject}` | `plugins` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/plugins/{idProject}/{plugin}` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/plugins/{idProject}/{plugin}` | `plugins` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/plugins/{idProject}/{step}` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/profile` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/profile` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/profile/mfa/confirm` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/profile/mfa/enroll` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/profile/mfa/step-up` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects` | `projects` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects` | `projects` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/create` | `projects` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-impact/{assetType}/{assetId}` | `asset-versions` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetType}/{assetId}` | `asset-versions` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}` | `asset-versions` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}/review-events` | `asset-versions` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{fromVersion}/diff/{toVersion}` | `asset-versions` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integration-deliveries` | `integrations` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/integration-deliveries/{integrationDelivery}/replay` | `integrations` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integrations` | `integrations` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/integrations` | `integrations` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/rotate-secret` | `integrations` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/status` | `integrations` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/test` | `integrations` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/matrix` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/results` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | `artifacts` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | `artifacts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | `artifacts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | `artifacts` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/impact` | `artifacts` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | `artifacts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | `artifacts` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/projects/{project}` | `projects` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{project}` | `projects` | `ready` | `go` | `send-to-go` |
| `PUT|PATCH` | `/api/admin/projects/{project}` | `projects` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/projects/{project}/edit` | `projects` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/result-exports` | `result-exports` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}` | `result-exports` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}/download` | `result-exports` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/roles` | `access-control` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/service-accounts` | `service-accounts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/service-accounts` | `service-accounts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/service-accounts/{serviceAccount}/revoke` | `service-accounts` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/steps` | `steps` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/steps/{idProject}` | `steps` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/steps/{idProject}/updateorder` | `steps` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/steps/{idProject}/{environment}` | `steps` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/steps/{idProject}/{step}` | `steps` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/steps/{idProject}/{step}` | `steps` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/stepsperfomed/{idTestPerformed}` | `execution-results` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/testcycles` | `test-cycles` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}` | `test-cycles` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}/{testcycle}` | `test-cycles` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/testcycles/{idProject}/{testcycle}` | `test-cycles` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/testcyclesperfomed/{idTestCyclePerformed}` | `execution-results` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/tests` | `tests` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/tests/{idProject}` | `tests` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/tests/{idProject}/{test}` | `tests` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/admin/tests/{idProject}/{test}` | `tests` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/testsperfomed/{idTestPerformed}` | `execution-results` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/audit-events` | `audit-events` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/clear` | `operations` | `laravel-operational` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/csrf-cookie` | `operations` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/agents/register` | `agent-registry` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/environment/{idEnvironment}` | `environments` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/environments/{idProject}` | `environments` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/plugin/{idPlugin}` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/plugins/{idProject}` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/matrix` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/results` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens/{tokenId}/revoke` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/step` | `cli-performed-steps` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/ideliumcl/step` | `cli-performed-steps` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/step/{idStep}` | `steps` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/test` | `cli-performed-tests` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/ideliumcl/test` | `cli-performed-tests` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/test/{idTest}` | `tests` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/testcycle` | `cli-performed-cycles` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/ideliumcl/testcycle` | `cli-performed-cycles` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/testcycle/{idTestCycle}` | `test-cycles` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumrunner/claim` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumrunner/heartbeat` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/ideliumrunner/worker` | `parallel-runs` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/login` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/logout` | `browser-identity` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/me/capabilities` | `access-control` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/menu/header` | `customers` | `ready` | `go` | `send-to-go` |
| `PUT` | `/api/menu/header/{idCostumer}` | `customers` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/menu/sidebar` | `access-control` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/oidc/token-exchange` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/sanctum/csrf-cookie` | `operations` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/sso/{identityProvider}/oidc/callback` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/sso/{identityProvider}/saml/callback` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/sso/{identityProvider}/start` | `enterprise-identity` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/user` | `browser-identity` | `ready` | `go` | `send-to-go` |

## Regeneration

```sh
python3 scripts/build_staging_route_cutover.py
python3 scripts/build_staging_route_cutover.py --check
```
