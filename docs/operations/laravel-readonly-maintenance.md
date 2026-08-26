# Laravel Read-only Maintenance

Issue [#175](https://github.com/ifuschini/idelium-api-go/issues/175)
defines the time-boxed Laravel read-only maintenance period used immediately
before archival. The Go service does not toggle Laravel into maintenance by
itself; instead, this gate defines the operational controls that must be true
before release operators enter the approved window.

## Generated evidence

The maintenance plan is generated from:

- `docs/contracts/laravel-schema-freeze.json`
- `docs/contracts/staging-route-cutover.json`

Run:

```sh
python3 scripts/build_laravel_readonly_maintenance.py
python3 scripts/build_laravel_readonly_maintenance.py --check
```

The generated outputs are:

- `docs/contracts/laravel-readonly-maintenance.json`
- `docs/contracts/laravel-readonly-maintenance.md`

`make verify` runs the maintenance check.

## Maintenance controls

| Control | Purpose |
| --- | --- |
| Gateway mutation block | Prevent Laravel-owned mutation traffic during the read-only window. |
| Queue drain | Drain Laravel queues before read-only mode and keep workers stopped. |
| Scheduled job pause | Pause Laravel jobs that can mutate persisted data. |
| Go route verification | Confirm Go-owned routes and explicit Go fail-closed gates before entering the window. |
| Operator broadcast | Announce the approved time-boxed maintenance window and expected user impact. |

## Current state

The current plan is intentionally `blocked` while route cutover blockers remain.
The maintenance window must not be scheduled until the staging route cutover
manifest reports zero Laravel blocker routes.

## Rollback

Rollback is operational and route-level:

1. Remove the gateway mutation block.
2. Resume Laravel workers.
3. Resume Laravel scheduled jobs.
4. Route traffic back to Laravel before retrying handover.

No database restore is required by this ticket because it does not apply schema
changes, enable Go writers, or introduce dual writes.

## Safety notes

- The maintenance plan records only aggregate statuses and control names.
- No tenant data, credentials, request payloads, cookies, or authorization
  headers are written to the plan.
- The read-only window is capped at 60 minutes and requires an approved start
  and end.
