# Staging Route Cutover Gate

Issue [#173](https://github.com/ifuschini/idelium-api-go/issues/173) defines
the staging control used before moving all remaining Laravel route ownership to
Go. The cutover is intentionally evidence-driven: a route may move in staging
only when it is implemented in Go or when Go exposes an explicit fail-closed
gate with a stable diagnostic.

## Generated evidence

The route decisions are generated from:

- `docs/contracts/migration-ownership-matrix.json`
- `docs/contracts/gateway-route-ownership.json`
- `api/openapi.yaml`

Generate or verify the manifest with:

```sh
python3 scripts/build_staging_route_cutover.py
python3 scripts/build_staging_route_cutover.py --check
```

The generated outputs are:

- `docs/contracts/staging-route-cutover.json`
- `docs/contracts/staging-route-cutover.md`

`make verify` runs the check target, so any stale staging cutover evidence fails
CI.

## Route states

| State | Staging owner | Action |
| --- | --- | --- |
| `ready` | `go` | Route can be sent to the Go service in staging. |
| `gated` | `go-fail-closed` | Route can be sent to the Go service only to confirm the expected fail-closed diagnostic. |
| `blocked` | `laravel` | Route remains on Laravel until a Go implementation or fail-closed gate is merged. |

## Safety policy

- Production cutover is disabled while any blocker exists.
- Application-level dual writes are prohibited.
- Laravel remains the fallback owner during staging rehearsals.
- The manifest must not contain credentials, authorization headers, cookies, or
  runtime token values.
- Gated routes must expose stable error codes such as
  `IDENTITY_MIGRATION_DISABLED` or `SERVICE_ACCOUNT_MIGRATION_DISABLED`.

## Rollback

Rollback remains route-level. If a staging rehearsal fails, route ownership
returns to Laravel at the gateway before any Go writer is enabled. Because this
gate does not enable dual writes or destructive schema changes, rollback does
not require database restoration.

## Production exit criteria

Production cutover can be proposed only when:

1. `docs/contracts/staging-route-cutover.json` reports `status: ready`.
2. `laravel_blocker_routes` is `0`.
3. Every mutation aggregate has one explicit owner.
4. The gateway configuration covers every Go-owned route.
5. `make verify` passes.
6. Rollback and monitoring evidence has been reviewed.
