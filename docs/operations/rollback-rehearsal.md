# Rollback Rehearsal

Issue [#177](https://github.com/ifuschini/idelium-api-go/issues/177)
defines the rehearsal for rolling back to the last dual-runtime release.

The rehearsal is a prerequisite for final schema handover and Laravel archival.
It proves that Idelium can return route ownership and the default API image to
Laravel without database restore, reverse application replay, or dual writes.

## Generated evidence

The rehearsal plan is generated from:

- `docs/contracts/docker-default-image-switch.json`
- `docs/contracts/laravel-readonly-maintenance.json`
- `docs/contracts/staging-route-cutover.json`

Run:

```sh
python3 scripts/build_rollback_rehearsal.py
python3 scripts/build_rollback_rehearsal.py --check
```

The generated outputs are:

- `docs/contracts/rollback-rehearsal.json`
- `docs/contracts/rollback-rehearsal.md`

`make verify` runs the rehearsal check.

## Ordered rehearsal

1. Freeze forward rollout and assign rollback ownership.
2. Switch route ownership back to Laravel at the gateway.
3. Restore the last dual-runtime Laravel API image default.
4. Resume Laravel queue workers and scheduled jobs.
5. Run Docker quickstart, Web smoke, CLI smoke, and route ownership checks.
6. Record command output, route owner, image digest, and observation window
   results.

## Success criteria

- Gateway owner after rollback is Laravel.
- Laravel API image is the default API image.
- Laravel workers and scheduled jobs are resumed.
- Go routes can be drained without reverse data replay.
- Docker quickstart, Web smoke, and CLI smoke checks pass.
- No database restore is required.

## Current state

The current plan is `blocked` while the Docker default switch, Laravel read-only
maintenance, and staging route cutover gates are not ready. The rehearsal should
not be treated as complete until those gates report `ready` and the ordered
steps have been executed in the target deployment topology.

## Safety

- Database restore is not part of the rollback strategy.
- Reverse application replay is forbidden.
- Dual writes are forbidden.
- The recovery objective is 30 minutes or less.
- The plan records only control states, ordered actions, and aggregate route
  counts. It contains no credentials, cookies, authorization headers, or payload
  data.
