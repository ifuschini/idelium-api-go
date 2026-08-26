# Final Operations Handover

This generated pack summarizes backup, recovery, release, and operations
requirements for the Laravel-to-Go API handover.

## Status

| Field | Value |
| --- | --- |
| Handover status | `blocked` |
| Production enabled | `false` |

## Gate statuses

| Gate | Status |
| --- | --- |
| `schema_freeze` | `frozen` |
| `route_cutover` | `blocked` |
| `laravel_readonly_maintenance` | `blocked` |
| `docker_default_image_switch` | `blocked` |
| `rollback_rehearsal` | `blocked` |

## Backup scope

- application database snapshot
- object-storage artifacts and retention metadata
- current Laravel API image digest
- candidate Go API image digest
- gateway route ownership configuration

## Recovery

- Route switchback owner: `laravel`
- Database restore required for route rollback: `false`
- Reverse application replay allowed: `false`
- Max recovery objective: 30 minutes

## Release

- Docker default target: `idelium/api-go`
- Image reference policy: `pin-by-immutable-digest`
- Readiness path: `/readyz`

Required smoke checks:
- `docker-quickstart`
- `web-smoke`
- `cli-smoke`
- `route-cutover-check`

## Operations

- Maintenance window cap: 60 minutes
- Maintenance controls:
  - `gateway-mutation-block`
  - `queue-drain`
  - `scheduled-job-pause`
  - `go-route-verification`
  - `operator-broadcast`
- Rollback steps:
  - `freeze-forward-rollout`
  - `gateway-switchback`
  - `restore-laravel-api-image`
  - `resume-laravel-processing`
  - `smoke-and-observe`
  - `record-rehearsal-evidence`

## Current blockers

| Gate | Status |
| --- | --- |
| `route_cutover` | `blocked` |
| `laravel_readonly_maintenance` | `blocked` |
| `docker_default_image_switch` | `blocked` |
| `rollback_rehearsal` | `blocked` |

## Regeneration

```sh
python3 scripts/build_operations_handover.py
python3 scripts/build_operations_handover.py --check
```
