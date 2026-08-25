# Gateway Route Ownership

This file records the Wave 3 gateway routing intent for platform catalog reads.
It complements [`route-rollout-overrides.json`](route-rollout-overrides.json):
the rollout override says which public Laravel route is Go-owned, while this
gateway contract says how a gateway should route that public path.

## Upstreams

- Go upstream: `IDELIUM_API_GO_BASE_URL`
- Laravel fallback upstream: `IDELIUM_LARAVEL_BASE_URL`

The public contract keeps the `/api` prefix. The Go service exposes the same
operation without the `/api` gateway prefix, for example:

| Public path | Go path |
| --- | --- |
| `/api/admin/platforms/types` | `/admin/platforms/types` |
| `/api/admin/platforms/status` | `/admin/platforms/status` |
| `/api/admin/platforms/browsers/{idOs}` | `/admin/platforms/browsers/{idOs}` |

Only `GET` and `HEAD` are routed to Go in this slice. Platform catalog
mutations remain Laravel-owned until the Wave 6 mutation aggregate.

## Rollback

Rollback is route-level and does not require database restore or dual writes:

1. Switch each route in
   [`gateway-route-ownership.json`](gateway-route-ownership.json) from `go` to
   `laravel`, or remove the Go-owned route entries.
2. Regenerate or reload the gateway configuration.
3. Verify the same public `/api/admin/platforms/...` paths respond from Laravel.

The legacy `browserversions` spelling is intentionally preserved because it is
part of the existing public route contract.
