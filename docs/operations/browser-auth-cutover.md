# Go browser-auth cutover

Issues #159 through #163 introduce the Go-native implementation for the
browser-auth bootstrap, current-user, menu, tenant-switch, account, role,
profile, and customer-administration endpoints. The gateway remains
Laravel-owned until the dependent browser routes are migrated, so a browser
never receives a Go session that a Laravel-owned protected route could
misinterpret.

Issues #136 through #138 add the first browser-authoring slices on top of those
sessions: test reads and writes, step membership through the test `config`
payload, Postman/Idelium import into steps and tests, test-cycle reads and
writes, and step ordering for a tenant-owned project. The routes stay compatible
with the existing Laravel Web client while enforcing the active Go session
tenant on every project, test, test-cycle, and step lookup.

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
| `GET /api/admin/roles` | `/admin/roles` | Role list filtered by the authenticated user's legacy role |
| `GET /api/admin/profile` | `/admin/profile` | Authenticated profile projection without password data |
| `PUT /api/admin/profile` | `/admin/profile` | Password update with Idelium password policy and bcrypt hashing |
| `GET /api/admin/accounts` | `/admin/accounts` | Tenant-scoped account grid/list without password data |
| `POST /api/admin/accounts` | `/admin/accounts` | Tenant-scoped account creation with password policy and bcrypt hashing |
| `PUT /api/admin/accounts/{idUser}` | `/admin/accounts/{idUser}` | Tenant-scoped account name/password update |
| `DELETE /api/admin/accounts/{idUser}` | `/admin/accounts/{idUser}` | Tenant-scoped account deletion |
| `GET /api/admin/costumers` | `/admin/costumers` | Superadmin customer grid/list without API keys |
| `POST /api/admin/costumers` | `/admin/costumers` | Superadmin customer creation with generated legacy API key |
| `PUT /api/admin/costumers/{idCostumer}` | `/admin/costumers/{idCostumer}` | Superadmin customer name/description update |
| `DELETE /api/admin/costumers/{idCostumer}` | `/admin/costumers/{idCostumer}` | Superadmin customer deletion |
| `GET /api/admin/tests/{idProject}` | `/admin/tests/{idProject}` | Tenant-scoped test grid/list for a project |
| `POST /api/admin/tests` | `/admin/tests` | Tenant-scoped test creation with a step-membership config and asset-version snapshot |
| `GET /api/admin/tests/{idProject}/{test}` | `/admin/tests/{idProject}/{test}` | Tenant-scoped test detail including config |
| `PUT /api/admin/tests/{idProject}/{test}` | `/admin/tests/{idProject}/{test}` | Tenant-scoped test config update with an asset-version snapshot |
| `POST /api/admin/importtest` | `/admin/importtest` | Transactional tenant-scoped import of Idelium/Postman steps into a generated test |
| `GET /api/admin/testcycles/{idProject}` | `/admin/testcycles/{idProject}` | Tenant-scoped test-cycle grid/list for a project |
| `POST /api/admin/testcycles` | `/admin/testcycles` | Tenant-scoped test-cycle creation with an asset-version snapshot |
| `GET /api/admin/testcycles/{idProject}/{testcycle}` | `/admin/testcycles/{idProject}/{testcycle}` | Tenant-scoped test-cycle detail |
| `PUT /api/admin/testcycles/{idProject}/{testcycle}` | `/admin/testcycles/{idProject}/{testcycle}` | Tenant-scoped test-cycle description/config update with an asset-version snapshot |
| `POST /api/admin/steps/{idProject}/updateorder` | `/admin/steps/{idProject}/updateorder` | Transactional tenant-scoped step order update |

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
Account administration requires the `accounts.manage` capability. Role 1 can
manage all accounts; role 2 can manage only non-superadmin accounts in the active
tenant; role 3 receives a forbidden response. Passwords are never returned,
stored in plaintext, or logged.
Customer administration requires the `customers.manage` capability, available to
role 1 only. Customer list responses do not expose legacy API keys; creation
generates a fresh opaque key and sets a one-year license expiration for
compatibility with the Laravel controller.
Test, import, test-cycle, and step-order routes require an authenticated Go
browser session, resolve the active tenant from that session, and constrain all
reads and writes by `idCostumer` and `idProject`. Cross-tenant projects, tests,
test cycles, and steps are returned as missing resources. Test and test-cycle
create and update writes append redacted `asset_versions` snapshots; raw session
cookies, CSRF values, API keys, and password material are never included in
those snapshots. Imports validate that the submitted `import` field is a
non-empty JSON array of Idelium steps, that every imported step has a name and
executable actions, and that Postman-marked steps contain a
`postman_collection` action with a collection payload. The import transaction
creates all generated steps with the legacy high sort order and commits the
generated test only after every step insert succeeds.

## Cutover and rollback

Before moving any route to Go, deploy the additive Laravel migration and verify
the unmodified Web login, reload, logout, expiry, import, test list/create/show/
update, test-cycle list/create/show/update, and step-order flows against a
Go-only staging route map. Move login, logout, CSRF, current-user,
capabilities, and the Go-native authenticated consumers in one gateway release;
there are no dual writes.

Rollback switches those route owners back to Laravel. Go sessions intentionally
do not deserialize in Laravel, so rollback invalidates Go sessions and requires
users to sign in again. Imports, test create/update, test-cycle create/update,
and step reordering use the existing Laravel tables, so rollback does not
require a database restore; new `asset_versions` rows are append-only
audit/history records. No credential sharing or auth bridge is required.
