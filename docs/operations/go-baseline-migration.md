# Go baseline migration

Wave 10 starts schema handover by producing a reviewed baseline for the complete
Laravel-owned schema. This slice is intentionally review-only: it records the
Laravel migration source set, aggregate checksum, handover policy, and rollback
guardrails without applying SQL or moving traffic.

## Baseline source

- Source runtime: `idelium-api` Laravel migrations.
- Source directory: `../idelium-api/database/migrations`.
- Reviewed baseline manifest:
  [`docs/contracts/go-baseline-migration.json`](../contracts/go-baseline-migration.json).
- Human-readable review document:
  [`docs/contracts/go-baseline-migration.md`](../contracts/go-baseline-migration.md).
- Embedded runtime manifest:
  [`internal/migrations/baseline_manifest.json`](../../internal/migrations/baseline_manifest.json).

The manifest records file names, file sizes, per-migration SHA-256 values, and
an aggregate checksum. It does not contain tenant data, request payloads,
credentials, authorization headers, cookies, session identifiers, or secrets.

## Safe plan command

Operators can inspect the currently embedded baseline plan without touching the
database:

```sh
go run ./cmd/migrate --plan
```

The command emits safe JSON with:

- baseline ID;
- migration count;
- aggregate checksum;
- review status;
- current handover policy.

The command does not apply the baseline. Application remains disabled until the
bridge, empty-install, upgrade, route cutover, and rollback rehearsal tickets
pass.

## Verification

Run:

```sh
make baseline-migration-check
make verify
```

`baseline-migration-check` rebuilds the manifest from the Laravel migration
source tree and fails if any reviewed artifact is stale.

## Ownership, compatibility, and rollback

Laravel remains the schema owner during coexistence. Go owns only the reviewed
baseline manifest in this slice. There are no dual writes, no schema changes, no
traffic changes, and no database side effects.

Rollback is a normal Git revert before schema handover. After the later handover
ticket applies Go migrations, rollback must follow the documented last
dual-runtime release rehearsal and backup/restore procedure.
