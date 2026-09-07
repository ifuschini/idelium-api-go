# CLI and Runner Smoke Matrix

Issue [#190](https://github.com/ifuschini/idelium-api-go/issues/190) is covered
by the pinned Docker captures and the Laravel feature suite. Each route is
executed with synthetic credentials injected at runtime; generated fixtures do
not contain tokens, cookies, or authorization headers.

| Flow | Laravel source | Go evidence |
| --- | --- | --- |
| CLI list/create/matrix/show | `idelium-api/tests/Feature/ParallelRunScheduleApiTest.php` | `testdata/golden/laravel-parallel-runs/cli-*.fixture.json` |
| Run-token issue, consume once, revoke, expiry, wrong agent | `idelium-api/tests/Feature/RunTokenTest.php` | `internal/browserauth/handler_test.go`, MySQL integration tests |
| Runner claim and worker-token propagation | `idelium-api/tests/Feature/RunTokenTest.php` | `testdata/golden/laravel-parallel-runs/runner-claim-go.fixture.json` |
| Worker heartbeat/update with valid token | `idelium-api/tests/Feature/ParallelRunScheduleApiTest.php` | `runner-heartbeat-go.fixture.json`, `runner-update-go.fixture.json` |
| Missing, invalid, expired, and foreign-tenant worker token | `idelium-api/tests/Feature/RunTokenTest.php` | pinned Docker HTTP negative smoke and MySQL binding checks |
| Cancellation and results | `idelium-api/tests/Feature/ParallelRunScheduleApiTest.php` | `browser-cancel-go.fixture.json`, `cli-cancel-go.fixture.json`, `results-go.fixture.json` |

The Laravel and Go fixture pairs are compared by
`scripts/compare_parallel_run_fixtures.py`; the current run compares 21
standard route pairs plus the browser-cancel and CLI-cancel pairs (23 total)
after deterministic timestamp/status/worker normalization.

## Security invariants

- Run tokens are one-time and short-lived; worker tokens are separate,
  bcrypt-hashed, expiring, and bound to tenant/project/run/worker.
- Invalid or foreign credentials fail with the documented denial status.
- Secret values are never printed, persisted in fixtures, or serialized in
  public worker state.

Verification runs in the pinned Go/MariaDB Compose profile and is destroyed
after capture; Laravel feature tests use its pinned Composer Docker profile.
