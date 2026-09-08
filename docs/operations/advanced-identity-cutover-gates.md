# Advanced identity cutover gates

Wave 9 covers the late identity surfaces that must not be partially enabled:

- identity provider administration;
- SCIM user lifecycle writes;
- MFA enrollment, confirmation, and step-up;
- workload OIDC token exchange;
- SSO bootstrap and OIDC/SAML callbacks;
- break-glass account controls.

This migration slice makes those routes explicit in the Go router and fails
closed with a stable `409` ownership response until Go-native identity is enabled. The
guard exists to avoid accidental `404` ambiguity or unsafe fallback behavior
when traffic is pointed at Go before the final compatibility gates pass.

## Current behavior

The following routes still return `IDENTITY_LARAVEL_OWNER` from Go:

- `GET /api/admin/identity/providers`
- `POST /api/admin/identity/providers`
- `PUT /api/admin/identity/accounts/{user}/break-glass`
- `POST /api/admin/identity/accounts/{user}/break-glass/test`
- `POST /api/admin/identity/providers/{identityProvider}/scim/users`
- `POST /api/admin/profile/mfa/enroll`
- `POST /api/admin/profile/mfa/confirm`
- `POST /api/admin/profile/mfa/step-up`
- `POST /api/sso/{identityProvider}/start`
- `POST /api/sso/{identityProvider}/oidc/callback`
- `POST /api/sso/{identityProvider}/saml/callback`

The response envelope follows the standard API error contract and includes a
correlation ID. The handlers do not read or echo callback payloads, assertions,
SAML documents, OIDC tokens, secrets, cookies, session identifiers, or
authorization headers.

`POST /api/oidc/token-exchange` is enabled as a fail-closed Go validator. It
requires the signed callback envelope and validates a compact JWT with
`HS256`, exact `iss`, `aud` (string or array), `exp`, `sub`, and nonce claims.
The one-time SSO state is consumed only after token validation. Configure the
verification inputs through secret-managed environment variables:

- `IDELIUM_OIDC_ISSUER`
- `IDELIUM_OIDC_AUDIENCE`
- `IDELIUM_OIDC_HS256_SECRET`

The signing key is never logged or returned. Production providers using
asymmetric keys/JWKS remain behind the cutover gate until their key discovery
and rotation policy is implemented.

## OpenAPI contract

The Laravel compatibility contracts remain the authoritative public contract
while Laravel owns these routes. The generated OpenAPI compatibility block also
marks these operations with:

- `x-idelium-go-cutover-gate: true`
- `x-idelium-go-cutover-error-code: "IDENTITY_LARAVEL_OWNER"`
- documented `409` ownership response

## Cutover requirements

Remove the cutover gate only after:

1. Go-native browser-auth and tenant resolution are active;
2. SSO/OIDC/SAML callback validation has replay-safe tests;
3. SCIM writes enforce tenant ownership in the same transaction;
4. workload identity exchange validates issuer, audience, expiry, nonce, and
   bound service account ownership;
5. MFA and break-glass operations emit redacted audit events;
6. Laravel-Go differential fixtures or explicit incompatibility decisions exist;
7. rollback to Laravel has been rehearsed.

## Rollback

Rollback remains route-level while Laravel is the owner. If a canary points one
of these routes to Go too early, restore the gateway route owner to Laravel. No
database migration or data repair is required because the Go gate performs no
writes.
