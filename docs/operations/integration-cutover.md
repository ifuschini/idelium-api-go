# Integration endpoint cutover

The Go service implements the Wave 7 browser API for integration endpoints,
delivery inspection, test-delivery creation, and replay. Every lookup and
mutation includes the active customer and project identifiers. Cross-tenant
resources are returned as not found.

## Secret compatibility

`APP_KEY` must contain the same 32-byte key used by Laravel, either directly or
with Laravel's `base64:` prefix. Go encrypts new and rotated endpoint secrets in
Laravel's authenticated AES-256-CBC envelope. Plaintext secrets are write-only,
are never serialized, and audit records contain `[REDACTED]`.

## Queue ownership gate

Test and replay requests persist a `pending` delivery in the existing
`integration_deliveries` table. The Go HTTP service does not publish a Laravel
serialized queue job. The Go dispatcher implements Laravel-compatible adapter
payloads, HMAC headers, bounded retry, dead-letter state, optimistic attempt
guards, and a second SSRF destination check. The `/idelium-worker` polling process
is disabled by default and acquires a global MySQL advisory lease when enabled.
Gateway ownership for mutation routes moves only after the issue #149 aggregate
drain verifier reports ready. This prevents dual consumption and abandoned jobs.

## Deployment and rollback

Before route cutover:

1. Verify the shared `APP_KEY` secret through its mounted secret file or runtime
   environment without printing it.
2. Complete the queue-drain evidence in issue #149.
3. Verify list, create, status, rotation, test, replay, and cross-tenant denial
   against staging.

Rollback is route-level: return all integration endpoint and delivery routes to
Laravel before resuming Laravel workers. No database rollback or reverse write
replay is required because Go uses the existing schema and Laravel-compatible
secret envelope. Never run Laravel and Go delivery consumers concurrently.
