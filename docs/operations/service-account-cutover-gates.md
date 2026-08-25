# Service account credential cutover gates

Issue [#165](https://github.com/ifuschini/idelium-api-go/issues/165) covers the
Wave 9 migration of service accounts and scoped credentials from Laravel to the
Go API.

The Go runtime now owns a safe fail-closed gate for the externally visible
service-account administration routes:

- `GET /api/admin/service-accounts`
- `POST /api/admin/service-accounts`
- `POST /api/admin/service-accounts/{serviceAccount}/revoke`

Until the complete credential lifecycle is implemented in Go, each route returns
HTTP `501` with the stable error code
`SERVICE_ACCOUNT_MIGRATION_DISABLED`.

## Why the route is gated

Service-account credentials require all of the following behaviors to migrate
together:

- browser-session authorization and customer ownership checks;
- scoped grant validation;
- credential hashing and one-time secret disclosure;
- expiration and revocation lifecycle management;
- last-used timestamps;
- audit events;
- redaction of names, secrets, tokens, and authorization material in diagnostics.

Partially migrating only one of these behaviors would risk inconsistent access
control or credential exposure. The gate therefore keeps the route visible to
contract tests while preventing writes or reads before the full cutover.

## Safety behavior

The gate does not deserialize request payloads and does not echo request body
fields. The response includes only:

- the stable migration-disabled error code;
- a safe client-facing message;
- the request correlation ID.

Logs include the route surface and correlation ID only. They do not include
credential values, scopes, request bodies, cookies, headers, session identifiers,
or tenant payload data.

## Cutover requirements

Remove the gate only after:

1. persistence schema compatibility is confirmed against the Laravel baseline;
2. service-account list, create, rotate, revoke, and last-used behavior has
   Laravel-compatible contract coverage;
3. tenant-scoped retrieval tests prove cross-customer credentials are hidden;
4. redaction tests cover secret material, scopes, tokens, and authorization
   headers;
5. rollback keeps Laravel as a valid route owner until production verification
   passes.

## Rollback

If service-account migration fails during staging or production rehearsal, keep
the route owner on Laravel and leave the Go fail-closed gate enabled. The gate is
safe because it prevents credential lifecycle writes and does not expose
credential payload data.
