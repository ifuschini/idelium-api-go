# HTTP Safety and Observability

This document defines the common HTTP middleware contract for every route
owned by `idelium-api-go`. Laravel remains the fallback owner for routes that
have not passed their migration gates.

## Correlation identifiers

Clients may send `X-Correlation-ID`. The API preserves it only when it contains
between 1 and 128 ASCII letters, digits, `.`, `_`, `:`, or `-`. Missing,
oversized, or malformed values are replaced with a generated identifier. The
validated value is:

- returned in the `X-Correlation-ID` response header;
- included in the stable JSON error envelope;
- attached to structured request and panic logs.

An invalid value is never reflected or logged. Correlation identifiers are
diagnostic values, not credentials or authorization boundaries.

## Structured request logs

One completion event records the validated correlation identifier, method,
URL path, response status, response byte count, and elapsed milliseconds. The
logger deliberately excludes query strings, request and response bodies,
headers, cookies, authorization values, session identifiers, credentials, and
tenant identifiers.

## Panic recovery

Unexpected panics are recovered at the HTTP boundary. Clients receive status
`500` with the stable `INTERNAL_ERROR` envelope and correlation identifier.
The raw panic value is not returned or logged. A server-side stack is recorded
with the safe request method, path, and correlation identifier for diagnosis.

## Response security headers

Every response receives the following baseline:

| Header | Value |
| --- | --- |
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'` |
| `Permissions-Policy` | `camera=(), geolocation=(), microphone=()` |
| `Referrer-Policy` | `no-referrer` |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-Permitted-Cross-Domain-Policies` | `none` |

Route-specific caching and CORS policies remain explicit contracts and are not
inferred by this middleware.

## Deployment and rollback

This middleware applies only to traffic already routed to Go and does not move
route ownership, access tenant data, or introduce writes. Rollback is the
revert of the dedicated middleware commit or routing affected paths back to
Laravel. Operators should retain correlation IDs when escalating an incident,
but must not attach raw credentials, headers, cookies, or payloads.

Laravel-Go differential comparison is not applicable to this ticket because it
adds the Go service safety envelope without moving a Laravel business route.
