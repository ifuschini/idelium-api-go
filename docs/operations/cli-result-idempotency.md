# CLI result idempotency

The Go-owned CLI result creation endpoints accept an optional `Idempotency-Key`
header:

- `POST /api/ideliumcl/testcycle`
- `POST /api/ideliumcl/test`
- `POST /api/ideliumcl/step`

Keys must contain 8 to 128 URL-safe letters, digits, hyphens, or underscores.
They are never logged or returned. A key is scoped by the authenticated
customer and result resource type. Repeating a successful request with the
same key returns the original result identifier and does not create another
row. The same key may be used by another customer without sharing a result.

The additive Laravel migration
`2026_08_27_120000_add_idempotency_keys_to_performed_results.php` must be
applied before routing these requests to Go. It adds nullable key columns and
tenant-scoped unique indexes, so legacy clients that omit the header retain
their existing behavior.

## Rollback

Route rollback returns the CLI write paths to Laravel. Keep the new nullable
columns and indexes in place; they are backward compatible and do not require
data replay or restoration. Do not enable automatic POST retries in clients
until every routed backend recognizes the header.
