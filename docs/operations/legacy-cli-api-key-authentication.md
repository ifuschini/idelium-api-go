# Legacy CLI API-key authentication

Wave 4 introduces the Go-side authentication boundary for legacy Idelium CLI
configuration reads without moving any `/api/ideliumcl` route ownership yet.
Laravel remains the fallback route owner until the CLI graph-read handlers and
their compatibility gates are complete.

## Public compatibility contract

Legacy CLI requests authenticate with the `Idelium-Key` HTTP header. Go preserves
the Laravel customer-key behavior:

- a missing, blank, malformed, expired, or unknown key returns HTTP `401`;
- the response body remains `{"message":"Invalid key"}` for compatibility with
  existing CLI diagnostics;
- a valid key resolves the customer tenant context before any tenant-owned
  resource is read;
- successful authentication updates `costumers.apiKeyLastUsedAt`;
- `costumers.apiKeyExpiresAt` is honored when present;
- service-account credentials are intentionally left to the later identity wave.

The middleware rejects duplicate `Idelium-Key` headers, keys with line breaks,
blank keys, and keys above the bounded parser limit before persistence lookup.

## Diagnostics and redaction

Authentication diagnostics are structured and deliberately exclude:

- the `Idelium-Key` header name and value;
- request headers, cookies, bodies, and query strings;
- repository error details that may contain connection strings or credentials.

Only safe metadata is logged: correlation identifier, method, path, and a bounded
rejection reason such as `missing_or_malformed` or `invalid_or_expired`.

## Tenant boundary

After successful authentication, handlers read the customer and tenant context
from the request context. Every future CLI configuration query must use that
customer identifier in the same query that reads the tenant-owned resource.
Cross-tenant references must continue to return Laravel-compatible `404`
responses rather than revealing whether the referenced resource exists.

## Deployment and rollback

This slice adds reusable Go authentication code only. It does not switch gateway
ownership for CLI routes and does not introduce dual writes. Rollback is the
normal strangler rollback: keep `/api/ideliumcl` traffic on Laravel and remove
the Go middleware from future CLI route groups if a compatibility issue is found.

Database compatibility depends on the existing Laravel columns:

- `costumers.apiKey`;
- `costumers.apiKeyExpiresAt`;
- `costumers.apiKeyLastUsedAt`.

No Go schema migration is introduced by this slice.
