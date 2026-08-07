# Laravel Route Consumer Map

This generated baseline maps current Idelium consumers to the Laravel routes
they invoke directly or through documented workflows. A missing consumer means
that the repository scan found no current usage; it does not authorize route
removal without compatibility review.

## Coverage summary

- Laravel routes: **171**
- Routes with an identified consumer: **107**
- Routes without an identified consumer: **64**

| Consumer | Mapped routes | Source baseline |
| --- | ---: | --- |
| `idelium-cli` | 13 | `34191d99c6f5c42118bf7622c4ec124251edeffa` |
| `idelium-docker` | 1 | `2b0f084d1cde906b36c2f6ce49dd90137edefa68` |
| `idelium-docker-wiki` | 13 | `59b406379c09ddbb66c21263fd4b477311c2e20e` |
| `idelium-runner` | 3 | `idelium-docker@2b0f084d1cde906b36c2f6ce49dd90137edefa68` |
| `idelium-web` | 91 | `23a21a7ce469bc4e0e417be42b0cfe089768a3bd` |

## Route-level mapping

| Method | Path | Authentication | Consumers |
| --- | --- | --- | --- |
| `GET|HEAD` | `/` | `public` | — |
| `POST` | `/_ignition/execute-solution` | `development-only` | — |
| `GET|HEAD` | `/_ignition/health-check` | `development-only` | — |
| `POST` | `/_ignition/update-config` | `development-only` | — |
| `GET|HEAD` | `/api/admin/accounts` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/accounts` | `browser-session` | `idelium-web` (direct) |
| `DELETE` | `/api/admin/accounts/{idUser}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/accounts/{idUser}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/agents` | `browser-session` | — |
| `PUT` | `/api/admin/agents/{agentRegistration}/status` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/apikey` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/apikey` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/costumers` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/costumers` | `browser-session` | `idelium-web` (direct) |
| `DELETE` | `/api/admin/costumers/{idCostumer}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/costumers/{idCostumer}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/environments` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/environments/{idProject}` | `browser-session` | `idelium-web` (direct) |
| `DELETE` | `/api/admin/environments/{idProject}/{environment}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/environments/{idProject}/{environment}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/environments/{idProject}/{environment}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/grid/bulk-jobs` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/grid/bulk-jobs/{jobId}/export` | `browser-session` | — |
| `POST` | `/api/admin/grid/query-snapshots` | `browser-session` | — |
| `PUT` | `/api/admin/identity/accounts/{user}/break-glass` | `browser-session` | — |
| `POST` | `/api/admin/identity/accounts/{user}/break-glass/test` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/identity/providers` | `browser-session` | — |
| `POST` | `/api/admin/identity/providers` | `browser-session` | — |
| `POST` | `/api/admin/identity/providers/{identityProvider}/scim/users` | `browser-session` | — |
| `POST` | `/api/admin/importtest` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/launchtest` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/brands` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/brands` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/brands` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/browsers` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/browsers` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/browsers/{idOs}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/browserversions` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/browserversions` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/browserversions/{idBrowser}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/locations` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/locations` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/locations` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/manageplatforms` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/manageplatforms` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/manageplatforms/{type}` | `browser-session` | `idelium-web` (direct) |
| `DELETE` | `/api/admin/platforms/manageplatforms/{type}/{id}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/models` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/models` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/models/{idBrand}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/os` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/os` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/os/{idType}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/platforms/osversion` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/platforms/osversion` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/osversion/{idOs}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/status` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/platforms/types` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/plugins` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/plugins/{idProject}` | `browser-session` | `idelium-web` (direct) |
| `DELETE` | `/api/admin/plugins/{idProject}/{plugin}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/plugins/{idProject}/{plugin}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/plugins/{idProject}/{step}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/profile` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/profile` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/profile/mfa/confirm` | `browser-session` | — |
| `POST` | `/api/admin/profile/mfa/enroll` | `browser-session` | — |
| `POST` | `/api/admin/profile/mfa/step-up` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/projects` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/create` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-impact/{assetType}/{assetId}` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetType}/{assetId}` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/asset-versions/{assetVersion}/review-events` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/asset-versions/{fromVersion}/diff/{toVersion}` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integration-deliveries` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/integration-deliveries/{integrationDelivery}/replay` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/integrations` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/integrations` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/rotate-secret` | `browser-session` | — |
| `PUT` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/status` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/integrations/{integrationEndpoint}/test` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/matrix` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/results` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/archive` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/delete-marker` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/impact` | `browser-session` | — |
| `PUT` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/legal-hold` | `browser-session` | — |
| `POST` | `/api/admin/projects/{idProject}/performed-test-cycles/{performedTestCycleId}/artifacts/{artifactDescriptor}/restore` | `browser-session` | — |
| `DELETE` | `/api/admin/projects/{project}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/{project}` | `browser-session` | `idelium-web` (direct) |
| `PUT|PATCH` | `/api/admin/projects/{project}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/projects/{project}/edit` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/result-exports` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/result-exports/{resultExport}/download` | `browser-session` | — |
| `GET|HEAD` | `/api/admin/roles` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/service-accounts` | `browser-session` | — |
| `POST` | `/api/admin/service-accounts` | `browser-session` | — |
| `POST` | `/api/admin/service-accounts/{serviceAccount}/revoke` | `browser-session` | — |
| `POST` | `/api/admin/steps` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/steps/{idProject}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/steps/{idProject}/updateorder` | `browser-session` | `idelium-web` (direct) |
| `DELETE` | `/api/admin/steps/{idProject}/{environment}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/steps/{idProject}/{step}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/steps/{idProject}/{step}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/stepsperfomed/{idTestPerformed}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/testcycles` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/testcycles/{idProject}/{testcycle}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/testcycles/{idProject}/{testcycle}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/testcyclesperfomed/{idTestCyclePerformed}` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/admin/tests` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/tests/{idProject}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/tests/{idProject}/{test}` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/admin/tests/{idProject}/{test}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/admin/testsperfomed/{idTestPerformed}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/audit-events` | `browser-session` | — |
| `GET|HEAD` | `/api/clear` | `public` | — |
| `GET|HEAD` | `/api/csrf-cookie` | `public` | — |
| `POST` | `/api/ideliumcl/agents/register` | `api-key` | — |
| `GET|HEAD` | `/api/ideliumcl/environment/{idEnvironment}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/environments/{idProject}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/plugin/{idPlugin}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/plugins/{idProject}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/matrix` | `api-key` | — |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/cancel` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/claim` | `api-key` | — |
| `GET|HEAD` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/results` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/tokens/{tokenId}/revoke` | `api-key` | — |
| `PUT` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}` | `api-key` | — |
| `POST` | `/api/ideliumcl/projects/{idProject}/parallel-runs/{parallelRun}/workers/{workerId}/heartbeat` | `api-key` | — |
| `POST` | `/api/ideliumcl/step` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `PUT` | `/api/ideliumcl/step` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/step/{idStep}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `POST` | `/api/ideliumcl/test` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `PUT` | `/api/ideliumcl/test` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/test/{idTest}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `POST` | `/api/ideliumcl/testcycle` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `PUT` | `/api/ideliumcl/testcycle` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `GET|HEAD` | `/api/ideliumcl/testcycle/{idTestCycle}` | `api-key` | `idelium-cli` (direct), `idelium-docker-wiki` (indirect-through-cli) |
| `POST` | `/api/ideliumrunner/claim` | `run-token` | `idelium-runner` (direct-contract) |
| `POST` | `/api/ideliumrunner/heartbeat` | `run-token` | `idelium-runner` (direct-contract) |
| `PUT` | `/api/ideliumrunner/worker` | `run-token` | `idelium-runner` (direct-contract) |
| `POST` | `/api/login` | `browser-auth-bootstrap` | `idelium-web` (direct) |
| `POST` | `/api/logout` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/me/capabilities` | `browser-session` | — |
| `GET|HEAD` | `/api/menu/header` | `browser-session` | `idelium-web` (direct) |
| `PUT` | `/api/menu/header/{idCostumer}` | `browser-session` | `idelium-web` (direct) |
| `GET|HEAD` | `/api/menu/sidebar` | `browser-session` | `idelium-web` (direct) |
| `POST` | `/api/oidc/token-exchange` | `workload-identity-exchange` | — |
| `GET|HEAD` | `/api/sanctum/csrf-cookie` | `browser-auth-bootstrap` | `idelium-web` (direct), `idelium-docker` (direct-operational-probe) |
| `POST` | `/api/sso/{identityProvider}/oidc/callback` | `sso-bootstrap-or-callback` | — |
| `POST` | `/api/sso/{identityProvider}/saml/callback` | `sso-bootstrap-or-callback` | — |
| `POST` | `/api/sso/{identityProvider}/start` | `sso-bootstrap-or-callback` | — |
| `GET|HEAD` | `/api/user` | `browser-session` | — |

## Consumer references without a registered Laravel route

These references are migration gaps, not active Laravel contracts.

| Consumer | Referenced path | Reason |
| --- | --- | --- |
| `idelium-web` | `/api/admin/launch` | Canonical launch endpoint referenced by the Web compatibility layer but not registered by Laravel. |
| `idelium-web` | `/api/admin/launch/preflight` | Canonical preflight endpoint referenced by the Web compatibility layer but not registered by Laravel. |
| `idelium-web` | `/api/admin/launch/targets` | Canonical launch-target endpoint referenced by the Web compatibility layer but not registered by Laravel. |
| `idelium-docker-wiki` | `/api/v1` | Planned versioned API prefix documented by the roadmap; no current Laravel route is registered under this prefix. |

## Governance

This documentation-only map moves no traffic and changes no schema. Rollback
is a Git revert. Before a route moves to Go, its mapped consumers must have a
versioned compatibility contract and differential tests. Unmapped routes require
an explicit retain, deprecate, or remove decision; absence of observed usage is
not sufficient evidence for deletion.

Regenerate this map after updating either the Laravel inventory or the
consumer rules:

```sh
python3 scripts/build_consumer_route_map.py \
  --inventory docs/contracts/laravel-routes.json \
  --rules docs/contracts/consumer-route-rules.json \
  --output-json docs/contracts/consumer-route-map.json \
  --output-markdown docs/contracts/consumer-route-map.md
```
