# Legacy API-key lifecycle cutover

The CLI legacy API-key authentication path is already owned by the Go API. The
browser-managed lifecycle routes remain gated until Go owns the browser session,
customer administration, tenant-scoped writes, and audit trail required to rotate
or disclose customer credentials safely.

## Current route ownership

| Method | Route | Current behavior | Stable error code |
| --- | --- | --- | --- |
| `GET` | `/admin/apikey` | Fails closed in Go until cutover. | `LEGACY_API_KEY_MIGRATION_DISABLED` |
| `PUT` | `/admin/apikey` | Fails closed in Go until cutover. | `LEGACY_API_KEY_MIGRATION_DISABLED` |

Laravel remains the fallback owner for the browser lifecycle flow until the Wave
9 exit criteria are complete. The Go gate does not perform dual writes and does
not reflect request payloads, API keys, expiration policy values, or credential
material in responses.

## Cutover requirements

Enable Go-native lifecycle management only after these gates are complete:

- browser-session authentication and CSRF protection are owned by Go;
- the selected customer is resolved from a trusted tenant-scoped session;
- key generation uses cryptographically secure randomness;
- stored key material is hashed or encrypted according to the final credential
  storage contract;
- expiration, creation time, last-used time, owner, and audit metadata are
  persisted with Laravel-compatible semantics;
- key replacement emits an auditable lifecycle event without logging secrets;
- rollback can route browser lifecycle requests back to Laravel without losing
  the previous valid key.

## Safety checks

- `PUT /admin/apikey` rejects credential payloads before reading or echoing them.
- Responses use the public error envelope and stable diagnostic code.
- Tests verify fail-closed status and redaction of submitted key material.
