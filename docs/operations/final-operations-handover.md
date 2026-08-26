# Final Operations Handover

Issue [#178](https://github.com/ifuschini/idelium-api-go/issues/178)
publishes the backup, recovery, release, and operations handover pack for the
Laravel-to-Go API migration.

## Generated evidence

The handover pack is generated from the Wave 10 gates:

- `docs/contracts/laravel-schema-freeze.json`
- `docs/contracts/staging-route-cutover.json`
- `docs/contracts/laravel-readonly-maintenance.json`
- `docs/contracts/docker-default-image-switch.json`
- `docs/contracts/rollback-rehearsal.json`

Run:

```sh
python3 scripts/build_operations_handover.py
python3 scripts/build_operations_handover.py --check
```

The generated outputs are:

- `docs/contracts/operations-handover.json`
- `docs/contracts/operations-handover.md`

`make verify` runs the handover check.

## Backup requirements

Before entering the final maintenance window, operators must capture:

- application database snapshot;
- object-storage artifacts and retention metadata;
- current Laravel API image digest;
- candidate Go API image digest;
- gateway route ownership configuration.

A restore test is required before the handover is considered ready.

## Recovery requirements

- Route switchback owner is Laravel.
- Route rollback must not require database restore.
- Reverse application replay is forbidden.
- Recovery objective is inherited from the rollback rehearsal and must remain
  30 minutes or less.

## Release requirements

- The target API image is `idelium/api-go`.
- The image reference must be pinned by immutable digest.
- `/readyz` must pass before traffic is admitted.
- Docker quickstart, Web smoke, CLI smoke, and route cutover checks must pass.

## Operations requirements

- The Laravel read-only maintenance window is capped at 60 minutes.
- Gateway mutation blocking, queue drain, scheduler pause, Go route verification,
  and operator broadcast controls must be confirmed.
- The rollback rehearsal steps must be executable in order.

## Current state

The handover remains blocked until all upstream gates are ready. This is
expected while route cutover blockers remain and production defaults continue to
fall back to Laravel.
