# Web Result Display Parity Contract

This contract defines the minimum result payload shape that Idelium Web must be
able to render for executions produced by the Go runtime. It is intentionally a
fixture-level contract while Wave 5 write routes and Wave 7 result exploration
routes remain Laravel-owned.

The machine-readable contract is
[`web-result-display-parity.json`](web-result-display-parity.json).

## Required views

- Test Running tab.
- Test Results tab.
- Cycle, run, test, and step selection states.
- Step-by-step execution detail.
- Per-step timing timeline, including the gap between steps.
- Environment, platform, browser, browser version, operating system, operating
  system version, and device summary.
- Postman request detail modal.
- Safe diagnostic messages.

## Fixtures

| Fixture | Purpose |
| --- | --- |
| `go-produced-browser-run` | Browser execution with passed and failed steps, timing, diagnostics, artifact summary, and platform metadata. |
| `go-produced-postman-run` | Postman execution with request-level status, assertions, timing, URL, and response summaries. |

## Compatibility policy

The fixtures use Laravel-compatible field names and public status values. They
do not move HTTP route ownership. Later route-migration tickets must either
produce these fields directly or document a versioned compatibility decision
before changing the Web contract.

## Redaction

The parity fixtures contain no credentials, cookies, authorization headers,
session identifiers, tenant-owned values, request bodies, response bodies, or raw
stack traces. Response body previews use `[REDACTED BODY]` unless a future ticket
adds an explicit safe-body display contract.
