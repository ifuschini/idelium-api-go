# Staging Route Cutover Manifest

This generated manifest is the staging checklist for moving route
ownership from Laravel to Go. It deliberately keeps production disabled
until every route is either implemented in Go or exposed as an explicit
Go fail-closed gate. Application-level dual writes remain prohibited.

## Status

| Field | Value |
| --- | --- |
| Cutover status | `blocked` |
| Production enabled | `false` |
| Route count | 168 |
| Go-owned routes | 17 |
| Go fail-closed routes | 15 |
| Laravel blocker routes | 136 |
| Gateway Go routes | 10 |

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
| `access-control` | 3 |
| `accounts` | 4 |
| `agent-registry` | 3 |
| `artifacts` | 7 |
| `asset-versions` | 5 |
| `audit-events` | 1 |
| `browser-identity` | 5 |
| `customers` | 6 |
| `environments` | 5 |
| `execution-results` | 9 |
| `grid-jobs` | 4 |
| `integrations` | 7 |
| `legacy-api-keys` | 2 |
| `operations` | 4 |
| `parallel-runs` | 23 |
| `platform-catalog` | 17 |
| `plugins` | 5 |
| `projects` | 7 |
| `result-exports` | 3 |
| `steps` | 6 |
| `test-cycles` | 4 |
| `test-launches` | 1 |
| `tests` | 5 |

## Route decisions

| Method | Path | Aggregate | State | Staging owner | Action |
| --- | --- | --- | --- | --- | --- |
| `GET|HEAD` | `/` | `operations` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/accounts` | `accounts` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/accounts` | `accounts` | `blocked` | `laravel` | `keep-on-laravel` |
| `DELETE` | `/api/admin/accounts/{idUser}` | `accounts` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/accounts/{idUser}` | `accounts` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/agents` | `agent-registry` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/agents/{agentRegistration}/status` | `agent-registry` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/apikey` | `legacy-api-keys` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/apikey` | `legacy-api-keys` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/costumers` | `customers` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/costumers` | `customers` | `blocked` | `laravel` | `keep-on-laravel` |
| `DELETE` | `/api/admin/costumers/{idCostumer}` | `customers` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/costumers/{idCostumer}` | `customers` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/environments` | `environments` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/environments/{idProject}` | `environments` | `blocked` | `laravel` | `keep-on-laravel` |
| `DELETE` | `/api/admin/environments/{idProject}/{environment}` | `environments` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/environments/{idProject}/{environment}` | `environments` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/environments/{idProject}/{environment}` | `environments` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/grid/bulk-jobs` | `grid-jobs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}` | `grid-jobs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}/export` | `grid-jobs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/grid/query-snapshots` | `grid-jobs` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/identity/accounts/{user}/break-glass` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/identity/accounts/{user}/break-glass/test` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `GET|HEAD` | `/api/admin/identity/providers` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/identity/providers` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/identity/providers/{identityProvider}/scim/users` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/importtest` | `tests` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/launchtest` | `test-launches` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/brands` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/brands` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/brands` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/platforms/browsers` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/browsers` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/browsers/{idOs}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/browserversions` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/browserversions` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/browserversions/{idBrowser}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/locations` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/locations` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/locations` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/platforms/manageplatforms` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/manageplatforms` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/manageplatforms/{type}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `DELETE` | `/api/admin/platforms/manageplatforms/{type}/{id}` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/platforms/models` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/models` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/models/{idBrand}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/os` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/os` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/os/{idType}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/platforms/osversion` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/platforms/osversion` | `platform-catalog` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/platforms/osversion/{idOs}` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/status` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/admin/platforms/types` | `platform-catalog` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/admin/plugins` | `plugins` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/plugins/{idProject}` | `plugins` | `blocked` | `laravel` | `keep-on-laravel` |
| `DELETE` | `/api/admin/plugins/{idProject}/{plugin}` | `plugins` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/plugins/{idProject}/{plugin}` | `plugins` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/plugins/{idProject}/{step}` | `plugins` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/profile` | `browser-identity` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/profile` | `browser-identity` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/profile/mfa/confirm` | `browser-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/profile/mfa/enroll` | `browser-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/profile/mfa/step-up` | `browser-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `GET|HEAD` | `/api/admin/projects` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/create` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-impact/{assetType}/{assetId}` | `asset-versions` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetType}/{assetId}` | `asset-versions` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}` | `asset-versions` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}/review-events` | `asset-versions` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{fromVersion}/diff/{toVersion}` | `asset-versions` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integration-deliveries` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/integration-deliveries/{integrationDelivery}/replay` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integrations` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/integrations` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/rotate-secret` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/status` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/test` | `integrations` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/matrix` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/results` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/impact` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | `artifacts` | `blocked` | `laravel` | `keep-on-laravel` |
| `DELETE` | `/api/admin/projects/{project}` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{project}` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT|PATCH` | `/api/admin/projects/{project}` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/projects/{project}/edit` | `projects` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/result-exports` | `result-exports` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}` | `result-exports` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}/download` | `result-exports` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/roles` | `access-control` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/service-accounts` | `service-accounts` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/service-accounts` | `service-accounts` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/service-accounts/{serviceAccount}/revoke` | `service-accounts` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/admin/steps` | `steps` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/steps/{idProject}` | `steps` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/steps/{idProject}/updateorder` | `steps` | `blocked` | `laravel` | `keep-on-laravel` |
| `DELETE` | `/api/admin/steps/{idProject}/{environment}` | `steps` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/steps/{idProject}/{step}` | `steps` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/steps/{idProject}/{step}` | `steps` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/stepsperfomed/{idTestPerformed}` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/testcycles` | `test-cycles` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}` | `test-cycles` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}/{testcycle}` | `test-cycles` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/testcycles/{idProject}/{testcycle}` | `test-cycles` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/testcyclesperfomed/{idTestCyclePerformed}` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/admin/tests` | `tests` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/tests/{idProject}` | `tests` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/tests/{idProject}/{test}` | `tests` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/admin/tests/{idProject}/{test}` | `tests` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/admin/testsperfomed/{idTestPerformed}` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/audit-events` | `audit-events` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/clear` | `operations` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/csrf-cookie` | `operations` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/agents/register` | `agent-registry` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/ideliumcl/environment/{idEnvironment}` | `environments` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/environments/{idProject}` | `environments` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/plugin/{idPlugin}` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/plugins/{idProject}` | `plugins` | `ready` | `go` | `send-to-go` |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/matrix` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/results` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens/{tokenId}/revoke` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumcl/step` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/ideliumcl/step` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/ideliumcl/step/{idStep}` | `steps` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/test` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/ideliumcl/test` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/ideliumcl/test/{idTest}` | `tests` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumcl/testcycle` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/ideliumcl/testcycle` | `execution-results` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/ideliumcl/testcycle/{idTestCycle}` | `test-cycles` | `ready` | `go` | `send-to-go` |
| `POST` | `/api/ideliumrunner/claim` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/ideliumrunner/heartbeat` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/ideliumrunner/worker` | `parallel-runs` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/login` | `browser-identity` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/logout` | `browser-identity` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/me/capabilities` | `access-control` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/menu/header` | `customers` | `blocked` | `laravel` | `keep-on-laravel` |
| `PUT` | `/api/menu/header/{idCostumer}` | `customers` | `blocked` | `laravel` | `keep-on-laravel` |
| `GET|HEAD` | `/api/menu/sidebar` | `access-control` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/oidc/token-exchange` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `GET|HEAD` | `/api/sanctum/csrf-cookie` | `operations` | `blocked` | `laravel` | `keep-on-laravel` |
| `POST` | `/api/sso/{identityProvider}/oidc/callback` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/sso/{identityProvider}/saml/callback` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `POST` | `/api/sso/{identityProvider}/start` | `enterprise-identity` | `gated` | `go-fail-closed` | `send-to-go-gate` |
| `GET|HEAD` | `/api/user` | `browser-identity` | `blocked` | `laravel` | `keep-on-laravel` |

## Regeneration

```sh
python3 scripts/build_staging_route_cutover.py
python3 scripts/build_staging_route_cutover.py --check
```
