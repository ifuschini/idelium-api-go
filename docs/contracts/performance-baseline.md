# Performance Baseline Report

This generated report identifies the representative read and write scenarios
that must be measured before migrated Go routes can replace Laravel-owned
behavior. The current baseline status is `capture-required` because this
repository does not contain live Laravel timing captures.

## Measurement Policy

- Environment: isolated non-production stack with synthetic tenants
- Clock: monotonic client-side duration around the HTTP request
- Redaction: No request or response payload values are written to the baseline report.
- Rollback: Performance evidence changes no traffic ownership and rolls back by Git revert.

## Representative Cases

| Class | Scenario | Route | Trust path | Consumers | p95 budget | Status |
| --- | --- | --- | --- | --- | ---: | --- |
| `read` | `platform-status-read` | `GET|HEAD /api/admin/platforms/status` | `browser-session` | `idelium-web` | 200 ms | `capture-required` |
| `read` | `platform-type-read` | `GET|HEAD /api/admin/platforms/types` | `browser-session` | `idelium-web` | 200 ms | `capture-required` |
| `read` | `cli-test-cycle-graph-read` | `GET|HEAD /api/ideliumcl/testcycle/{idTestCycle}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | 500 ms | `capture-required` |
| `read` | `cli-step-read` | `GET|HEAD /api/ideliumcl/step/{idStep}` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | 400 ms | `capture-required` |
| `read` | `web-test-result-detail-read` | `GET|HEAD /api/admin/testsperfomed/{idTestPerformed}` | `browser-session` | `idelium-web` | 650 ms | `capture-required` |
| `write` | `web-test-create-write` | `POST /api/admin/tests` | `browser-session` | `idelium-web` | 900 ms | `capture-required` |
| `write` | `cli-step-result-write` | `POST /api/ideliumcl/step` | `api-key` | `idelium-cli`, `idelium-docker-wiki` | 800 ms | `capture-required` |
| `write` | `web-launch-test-write` | `POST /api/admin/launchtest` | `browser-session` | `idelium-web` | 1200 ms | `capture-required` |

## Gate Policy

A route cannot move traffic to Go when its representative scenario exceeds
the p95 budget by more than 20% or records a non-zero error rate. Mutation
routes must also pass the side-effect comparator before performance evidence
can be accepted.

Regenerate this report with:

```sh
python3 scripts/build_performance_baseline.py \
  --backlog docs/contracts/compatibility-backlog.json \
  --output-json docs/contracts/performance-baseline.json \
  --output-markdown docs/contracts/performance-baseline.md
```
