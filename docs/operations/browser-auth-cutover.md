# Go browser-auth cutover

Issues #159 through #161 introduce the Go-native implementation for the
browser-auth bootstrap, current-user, menu, and tenant-switch endpoints. The
gateway remains Laravel-owned until the dependent browser routes are migrated,
so a browser never receives a Go session that a Laravel-owned protected route
could misinterpret.

## Contract

The Go runtime exposes these paths without the gateway `/api` prefix:

| Public path | Go path | Response |
| --- | --- | --- |
| `GET /api/sanctum/csrf-cookie` | `/sanctum/csrf-cookie` | `204` and a secure `XSRF-TOKEN` cookie |
| `POST /api/login` | `/login` | Laravel-compatible `authenticated` response and secure opaque session |
| `POST /api/logout` | `/logout` | `204`; rejects an absent session with `401` |
| `GET /api/user` | `/user` | Minimal authenticated user projection without password or session data |
| `GET /api/me/capabilities` | `/me/capabilities` | Versioned capability list for the authenticated user's role |
| `GET /api/menu/header` | `/menu/header` | Active-tenant projects, superadmin customer choices, and tenant context |
| `GET /api/menu/sidebar` | `/menu/sidebar` | Role-aware legacy navigation entries |
| `PUT /api/menu/header/{idCostumer}` | `/menu/header/{idCostumer}` | Superadmin tenant switch with reason, expiry, and audit event |

The session cookie is `HttpOnly`, `Secure`, `SameSite=Lax`, and expires after
120 minutes. The CSRF cookie is readable by the existing Web client and is
sent as `X-XSRF-TOKEN`. Raw session and CSRF values are never persisted: the
`go_browser_sessions` table stores SHA-256 hashes with the user and tenant IDs.
Password verification uses bcrypt against the existing Laravel user hash. Tenant
switching keeps the actor tenant immutable and stores the active tenant,
impersonation reason, and impersonation expiry on the Go session.
Current-user reads join the session tenant and user tenant in the same query,
hide expired, disabled, missing, and cross-tenant sessions as unauthenticated,
and clear Go-owned cookies when a stored session can no longer authenticate.
Tenant switches require role 1, a reason, a future expiry, and an existing target
tenant, then append a redacted `tenant.switch` audit event.

## Cutover and rollback

Before moving any route to Go, deploy the additive Laravel migration and verify
the unmodified Web login, reload, logout, and expiry flows against a Go-only
staging route map. Move login, logout, CSRF, current-user, capabilities, and
the next Go-native authenticated consumers in one gateway release; there are no
dual writes.

Rollback switches those route owners back to Laravel. Go sessions intentionally
do not deserialize in Laravel, so rollback invalidates Go sessions and requires
users to sign in again. No database restore, credential sharing, or auth bridge
is required.
