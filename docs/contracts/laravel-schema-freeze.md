# Laravel Schema Freeze

This generated report freezes the Laravel migration tree after the reviewed
Go baseline has been produced. It prevents accidental Laravel schema drift
during the final handover wave.

## Status

| Field | Value |
| --- | --- |
| Freeze status | `frozen` |
| Baseline ID | `go-baseline-2026-08-25` |
| Expected migrations | 66 |
| Current migrations | 66 |
| Expected aggregate SHA-256 | `7d37bf8e716cd62affe16c8f74b3f52fa02c8e284af5eb202e04465353b1bf5a` |
| Current aggregate SHA-256 | `7d37bf8e716cd62affe16c8f74b3f52fa02c8e284af5eb202e04465353b1bf5a` |

## Policy

- New Laravel migrations are not allowed during schema handover.
- Edits to reviewed Laravel migrations are not allowed.
- Go baseline application remains disabled until the bridge, empty-install,
  upgrade, route-cutover, and rollback rehearsal gates pass.
- Dual writes remain prohibited.
- Any exception must be handled as a reviewed, versioned schema-change
  issue that updates the Go baseline deliberately.

## Violations

| File | Type |
| --- | --- |
| none | none |

## Regeneration

```sh
python3 scripts/check_laravel_schema_freeze.py
python3 scripts/check_laravel_schema_freeze.py --check
```
