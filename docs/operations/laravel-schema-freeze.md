# Laravel Schema Freeze

Issue [#174](https://github.com/ifuschini/idelium-api-go/issues/174)
freezes the Laravel migration tree after the reviewed Go baseline has been
captured. This prevents accidental schema drift while `idelium-api-go` prepares
for final schema handover.

## Generated evidence

The freeze report is generated from the current Laravel migration directory and
the reviewed Go baseline manifest:

- Source migrations: `../idelium-api/database/migrations`
- Reviewed baseline: `docs/contracts/go-baseline-migration.json`
- Freeze report: `docs/contracts/laravel-schema-freeze.json`
- Human-readable report: `docs/contracts/laravel-schema-freeze.md`

Run:

```sh
python3 scripts/check_laravel_schema_freeze.py
python3 scripts/check_laravel_schema_freeze.py --check
```

`make verify` runs the freeze check, so CI fails if a Laravel migration is added,
removed, or modified without a deliberate baseline review.

## Policy

- New Laravel migrations are not allowed during schema handover.
- Edits to reviewed Laravel migrations are not allowed.
- Go baseline application remains disabled until the bridge, empty-install,
  upgrade, route-cutover, and rollback rehearsal gates pass.
- Dual writes remain prohibited.
- The freeze report records only file names, counts, sizes, and SHA-256 hashes.
  It does not contain tenant data, credentials, request payloads, cookies, or
  authorization headers.

## Exception process

If a schema change is unavoidable before handover:

1. Open a versioned schema-change issue describing the compatibility impact.
2. Prove the change is backward compatible for Laravel and Go coexistence.
3. Regenerate and review the Go baseline manifest.
4. Regenerate the Laravel schema freeze report.
5. Run `make verify`.
6. Document rollback and deployment order before merging.

## Rollback

This ticket does not apply migrations and does not move schema ownership. A
rollback is a Git revert of the freeze contract and verification target. Runtime
traffic and database ownership remain unchanged.
