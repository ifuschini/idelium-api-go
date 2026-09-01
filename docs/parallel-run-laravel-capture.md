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
