# Laravel parallel-run capture

The disposable Laravel capture profile uses `SESSION_DOMAIN=localhost`. Run
the capture runner against `http://localhost:<port>` rather than
`127.0.0.1`, otherwise the browser session cookie is rejected by the client.

The browser flow is:

1. `GET /api/sanctum/csrf-cookie`.
2. URL-decode the `XSRF-TOKEN` cookie and send it as `X-XSRF-TOKEN` and
   `X-CSRF-TOKEN` when posting to `/api/login`.
3. Preserve the post-login `laravel_session` and rotated `XSRF-TOKEN` values
   for subsequent browser requests.

Parallel-run claim, heartbeat, and worker mutation endpoints consume a
run-token. They must be captured with the dedicated runner authentication and
an issued synthetic token; a browser session alone is intentionally rejected.
Do not turn those routes into browser fixtures or bypass CSRF/authentication to
force a response.

All captured fixtures must be sanitized and validated with:

```sh
python3 scripts/validate_golden_fixtures.py <capture-directory>
```

## Go capture harness

The same runner can capture a Go owner by supplying a Go base URL and writing
the `-go` fixture variants. Use a real immutable source revision and synthetic
credentials only:

```sh
CAPTURE_LARAVEL_BASE_URL=http://localhost:18080 \
CAPTURE_LARAVEL_REVISION=$(git rev-parse HEAD) \
CAPTURE_API_KEY=fixture-cli-key-9001 \
python3 scripts/capture_parallel_runs.py \
  --plan testdata/parallel-run-capture-extended.json \
  --output-dir testdata/golden/laravel-parallel-runs \
  --runtime go \
  --repository idelium/idelium-api-go \
  --output-suffix -go
```

Browser routes require a synthetic Go browser session exported as
`CAPTURE_BROWSER_COOKIE` and `CAPTURE_XSRF_TOKEN`; runner routes additionally
require a short-lived `CAPTURE_RUN_TOKEN`. The harness never writes request
credentials and sanitizes response bodies before writing fixtures.

## Verified Docker capture

The Laravel and Go captures were executed against isolated, pinned Compose
profiles on 2026-09-10. The run covered all 21 routes in
`testdata/parallel-run-capture-extended.json`; credentials, cookies, and run
tokens are not persisted. Re-run validation and the differential comparator
after refreshing either runtime:

```sh
python3 scripts/validate_golden_fixtures.py testdata/golden/laravel-parallel-runs
python3 scripts/compare_parallel_run_fixtures.py \
  --plan testdata/parallel-run-capture-extended.json \
  --laravel-dir testdata/golden/laravel-parallel-runs \
  --go-dir testdata/golden/laravel-parallel-runs
```
