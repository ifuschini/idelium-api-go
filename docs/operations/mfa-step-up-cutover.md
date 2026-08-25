# MFA and step-up authentication cutover

MFA enrollment, confirmation, and step-up authentication remain gated until the
Go runtime owns browser authentication, tenant resolution, and identity audit
events. The routes are present in Go so premature gateway routing fails closed
with `IDENTITY_MIGRATION_DISABLED` rather than an ambiguous `404`.

## Guarded routes

- `POST /api/admin/profile/mfa/enroll`
- `POST /api/admin/profile/mfa/confirm`
- `POST /api/admin/profile/mfa/step-up`

The gate returns a standard API error envelope with a correlation ID and does
not inspect or echo request bodies. OTPs, recovery codes, session cookies,
authorization headers, and device identifiers are never written to the response
or logs by this guard.

## Enablement checklist

Remove the guard only after:

1. Go-native session and tenant resolution are active for browser requests;
2. MFA secrets are stored using the approved credential storage policy;
3. one-time passwords, recovery codes, and step-up challenges are validated with
   replay protection;
4. all success and failure paths emit redacted audit events;
5. cross-tenant tests prove that a user cannot enroll, confirm, or satisfy
   step-up for another customer;
6. rollback to Laravel has been rehearsed for these routes.

## Rollback

Before Go-native MFA writes are enabled, rollback is route-level: point the MFA
routes back to Laravel. The current guard does not write data, so no database
repair is required.
