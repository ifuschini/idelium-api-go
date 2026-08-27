# Go browser-auth cutover

Issue #159 introduces the Go-native implementation for the browser-auth
bootstrap endpoints. The gateway remains Laravel-owned until the dependent
current-user and authorization routes are migrated, so a browser never receives
a Go session that a Laravel-owned protected route could misinterpret.

## Contract

The Go runtime exposes these paths without the gateway `/api` prefix:

| Public path | Go path | Response |
| --- | --- | --- |
| `GET /api/sanctum/csrf-cookie` | `/sanctum/csrf-cookie` | `204` and a secure `XSRF-TOKEN` cookie |
| `POST /api/login` | `/login` | Laravel-compatible `authenticated` response and secure opaque session |
| `POST /api/logout` | `/logout` | `204`; rejects an absent session with `401` |

The session cookie is `HttpOnly`, `Secure`, `SameSite=Lax`, and expires after
120 minutes. The CSRF cookie is readable by the existing Web client and is
sent as `X-XSRF-TOKEN`. Raw session and CSRF values are never persisted: the
`go_browser_sessions` table stores SHA-256 hashes with the user and tenant IDs.
Password verification uses bcrypt against the existing Laravel user hash.

## Cutover and rollback

Before moving any route to Go, deploy the additive Laravel migration and verify
the unmodified Web login, reload, logout, and expiry flows against a Go-only
staging route map. Move login, logout, CSRF, and the Go-native authenticated
consumer in one gateway release; there are no dual writes.

Rollback switches those route owners back to Laravel. Go sessions intentionally
do not deserialize in Laravel, so rollback invalidates Go sessions and requires
users to sign in again. No database restore, credential sharing, or auth bridge
is required.
