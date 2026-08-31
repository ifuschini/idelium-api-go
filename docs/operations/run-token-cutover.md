# Parallel run token cutover

Issue #154 completes Go ownership of run-token issuance, validation,
single-use consumption, and revocation. Token administration remains available
only through the customer API-key routes; browser sessions do not gain a new
credential-management surface.

## Credential and isolation policy

- Issued credentials use the `idrt_<tokenId>.<secret>` compatibility format.
  The plaintext secret is returned once, stored only as a bcrypt hash, and never
  written to logs, audits, error responses, or documentation examples.
- Tokens default to a 300-second lifetime. A positive
  `IDELIUM_RUN_TOKEN_TTL_SECONDS` value preserves the Laravel deployment
  override. Claim enforcement remains enabled unless
  `IDELIUM_RUN_TOKEN_REQUIRED_FOR_CLAIM=false` is explicitly configured.
- Every issue, consume, validate, and revoke query scopes customer, project,
  schedule, agent, and token identifier as applicable. Missing and cross-tenant
  resources remain indistinguishable.
- Consumption locks the token row and sets `usedAt` once. Validation rejects a
  malformed, expired, used, revoked, wrongly scoped, wrongly bound, or
  bcrypt-mismatched credential with the Laravel validation envelope.
- Revocation locks the scoped token row and is idempotent: a repeated request
  preserves the first `revokedAt` value.
- Issue, successful consumption, rejection, and revocation audits contain only
  `[REDACTED]` token and token-id markers. Audit persistence remains fail-safe.

The Laravel `RunTokenTest` is the differential reference. Go handler tests cover
success, validation, not-found, and safe diagnostics; MySQL integration verifies
bcrypt storage, TTL, one-use and revoked-token rejection, idempotent revocation,
audit redaction, and negative cross-tenant access.

## Deployment and rollback

Deploy the pinned Go image after Go verification, MySQL integration, and the PHP
reference suite pass. Route the two API-key token administration operations and
the already migrated claim validation path to Go. Runner-only claim, heartbeat,
and worker-update routes remain Laravel-owned until issue #156.

Rollback routes token administration and claim validation back to Laravel.
Hashes, timestamps, token identifiers, and audit records remain mutually
compatible in the shared schema; no schema rollback, credential export, reverse
replay, or dual write is required. Never attempt to reconstruct a plaintext
secret during rollback.
