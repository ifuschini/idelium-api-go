# Docker Default API Image Switch

This generated plan governs the switch from the Laravel API image to the
Go API image as the default backend in `idelium-docker`. It deliberately
keeps production disabled until the read-only maintenance and staging
route cutover gates are ready.

## Status

| Field | Value |
| --- | --- |
| Switch status | `blocked` |
| Production enabled | `false` |
| Target API image | `idelium/api-go` |
| Image reference policy | `pin-by-immutable-digest` |
| Runtime user | `65532:65532` |
| Readiness path | `/readyz` |
| Maintenance status | `blocked` |
| Route cutover status | `blocked` |
| Laravel blocker routes | 134 |

## Switch controls

- Build and publish the exact Go API runtime image by immutable digest.
- Update idelium-docker defaults only after maintenance and route-cutover gates are ready.
- Keep the Laravel API image reference available as rollback fallback.
- Require Go readiness before web and CLI traffic is admitted.
- Run Docker quickstart, Web smoke, CLI smoke, and route cutover checks after the switch.

## Current blockers

| Control | Reason | Count |
| --- | --- | ---: |
| `laravel-readonly-maintenance` | The Laravel read-only maintenance gate is not ready. | 1 |
| `staging-route-cutover` | The staging route cutover manifest still has Laravel blockers. | 134 |

## Rollback

Restore the previous Laravel API image default, route traffic back to Laravel, and keep Go image available for diagnostics.

- Requires database restore: `false`
- Dual writes allowed: `false`

## Regeneration

```sh
python3 scripts/build_docker_default_image_switch.py
python3 scripts/build_docker_default_image_switch.py --check
```
