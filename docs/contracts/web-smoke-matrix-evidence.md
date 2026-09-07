# Parallel-run Web Smoke Matrix Evidence

Issue [#189](https://github.com/ifuschini/idelium-api-go/issues/189) is covered
by the isolated Docker profile. A deterministic synthetic browser session is
seeded in `smoke/seed.sql`; cookies and CSRF values are injected only at
runtime and removed before fixtures are written.

The browser matrix covers create, matrix, list, show, claim, results,
heartbeat, worker update, and cancel. Laravel fixtures and Go fixtures are
stored together under `testdata/golden/laravel-parallel-runs` and compared by
`scripts/compare_parallel_run_fixtures.py` (21 standard pairs plus the browser
cancel pair). Laravel feature coverage is maintained in
`idelium-api/tests/Feature/BrowserSessionAuthenticationTest.php` and
`ParallelRunScheduleApiTest.php`; Go coverage is in
`internal/browserauth/handler_test.go` and the real HTTP capture.

Verification also checks CSRF/session establishment, tenant ownership,
foreign-tenant denial, and redaction of cookie, CSRF, authorization, and
credential fields. The Docker profile is destroyed after capture.
