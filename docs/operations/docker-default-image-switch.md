# Docker Default API Image Switch

Issue [#176](https://github.com/ifuschini/idelium-api-go/issues/176)
tracks the switch that will make the Go API image the default backend image in
`idelium-docker`.

The current implementation is a guarded switch plan. It intentionally does not
change production defaults while the staging route cutover and Laravel read-only
maintenance gates are blocked.

## Generated evidence

The switch plan is generated from:

- `docs/contracts/laravel-readonly-maintenance.json`
- `docs/contracts/staging-route-cutover.json`

Run:

```sh
python3 scripts/build_docker_default_image_switch.py
python3 scripts/build_docker_default_image_switch.py --check
```

The generated outputs are:

- `docs/contracts/docker-default-image-switch.json`
- `docs/contracts/docker-default-image-switch.md`

`make verify` runs the switch-plan check.

## Target default

| Field | Value |
| --- | --- |
| API service image | `idelium/api-go` |
| Image reference policy | Pin by immutable digest |
| Runtime user | `65532:65532` |
| Readiness path | `/readyz` |
| Liveness path | `/healthz` |

## Switch controls

Before `idelium-docker` can make `idelium/api-go` the default API service:

1. Build and publish the exact Go API runtime image by immutable digest.
2. Confirm the Laravel read-only maintenance gate is ready.
3. Confirm the staging route cutover gate is ready.
4. Keep the Laravel API image reference available as rollback fallback.
5. Require Go readiness before web and CLI traffic is admitted.
6. Run Docker quickstart, Web smoke, CLI smoke, and route cutover checks.

## Current state

The plan remains `blocked` while Laravel route blockers exist. This prevents
`idelium-docker` from advertising the Go image as the default before the runtime
can satisfy all required route ownership evidence.

## Rollback

Rollback is image- and route-level:

1. Restore the previous Laravel API image default.
2. Route traffic back to Laravel.
3. Keep the Go image available for diagnostics.

No database restore is required by this ticket because it does not apply schema
changes or enable dual writes.
