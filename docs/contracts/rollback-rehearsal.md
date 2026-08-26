# Rollback Rehearsal Plan

This generated plan rehearses rollback to the last dual-runtime release.
It is intentionally blocked until the Docker default switch, Laravel
read-only maintenance, and staging route cutover gates are ready.

## Status

| Field | Value |
| --- | --- |
| Rehearsal status | `blocked` |
| Production enabled | `false` |
| Target release | `last-dual-runtime-release` |
| Gateway owner after rollback | `laravel` |
| Docker switch status | `blocked` |
| Maintenance status | `blocked` |
| Route cutover status | `blocked` |
| Laravel blocker routes | 136 |
| Max recovery objective | 30 minutes |

## Ordered rehearsal steps

| # | Step | Description |
| ---: | --- | --- |
| 1 | `freeze-forward-rollout` | Stop further Go promotion and declare rollback ownership. |
| 2 | `gateway-switchback` | Switch route ownership back to Laravel at the gateway. |
| 3 | `restore-laravel-api-image` | Restore the last dual-runtime Laravel API image default. |
| 4 | `resume-laravel-processing` | Resume Laravel queue workers and scheduled jobs after switchback. |
| 5 | `smoke-and-observe` | Run Docker quickstart, Web smoke, CLI smoke, and route ownership checks. |
| 6 | `record-rehearsal-evidence` | Record command output, route owner, image digest, and observation window results. |

## Success criteria

- Gateway owner after rollback is Laravel.
- Laravel API image is the default API image.
- Laravel workers and scheduled jobs are resumed.
- Go routes can be drained without reverse data replay.
- Docker quickstart, Web smoke, and CLI smoke checks pass.
- No database restore is required.

## Safety

- Requires database restore: `false`
- Reverse application replay allowed: `false`
- Dual writes allowed: `false`

## Current blockers

| Control | Reason |
| --- | --- |
| `docker-default-image-switch` | docker-default-image-switch is not ready. |
| `laravel-readonly-maintenance` | laravel-readonly-maintenance is not ready. |
| `staging-route-cutover` | staging-route-cutover is not ready. |

## Regeneration

```sh
python3 scripts/build_rollback_rehearsal.py
python3 scripts/build_rollback_rehearsal.py --check
```
