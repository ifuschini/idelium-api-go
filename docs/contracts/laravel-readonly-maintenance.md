# Laravel Read-only Maintenance Gate

This generated gate controls the time-boxed Laravel read-only period used
before archival. It does not move traffic by itself; it defines the
preconditions and operational controls that must be true before operators
enter the maintenance window.

## Status

| Field | Value |
| --- | --- |
| Maintenance status | `blocked` |
| Production enabled | `false` |
| Default state | `not-scheduled` |
| Max duration | 60 minutes |
| Schema freeze | `frozen` |
| Route cutover | `blocked` |
| Laravel blocker routes | 134 |
| Go-owned routes | 17 |
| Go fail-closed routes | 17 |

## Controls

| Control | Owner | State | Description |
| --- | --- | --- | --- |
| `gateway-mutation-block` | `gateway` | `planned` | Block Laravel-owned mutation traffic at the gateway during the approved window. |
| `queue-drain` | `laravel-operations` | `planned` | Drain Laravel queues before entering read-only and keep workers stopped during the window. |
| `scheduled-job-pause` | `laravel-operations` | `planned` | Pause Laravel scheduled jobs that can mutate data until archival is complete or rollback starts. |
| `go-route-verification` | `idelium-api-go` | `planned` | Verify Go-owned and Go fail-closed routes before read-only maintenance begins. |
| `operator-broadcast` | `release-management` | `planned` | Announce the time-boxed maintenance window and expected read-only behavior. |

## Exit criteria

- No Laravel schema freeze violations exist.
- No unresolved route cutover blockers exist.
- Laravel queues are drained and workers are stopped.
- Go-owned routes pass the staging smoke plan.
- Rollback owner and gateway switchback command are confirmed.

## Current blockers

| Control | Reason | Count |
| --- | --- | ---: |
| `route-cutover` | Routes without Go implementation or fail-closed gates remain on Laravel. | 134 |

## Rollback

Remove the gateway mutation block, resume Laravel workers and scheduled jobs, and route traffic back to Laravel before retrying the handover.

- Requires database restore: `false`
- Dual writes allowed: `false`

## Regeneration

```sh
python3 scripts/build_laravel_readonly_maintenance.py
python3 scripts/build_laravel_readonly_maintenance.py --check
```
