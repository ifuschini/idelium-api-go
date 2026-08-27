# Laravel Schema Freeze

This generated report freezes the Laravel migration tree after the reviewed
Go baseline has been produced. It prevents accidental Laravel schema drift
during the final handover wave.

## Status

| Field | Value |
| --- | --- |
| Freeze status | `frozen` |
| Baseline ID | `go-baseline-2026-08-25` |
| Expected migrations | 69 |
| Current migrations | 69 |
| Expected aggregate SHA-256 | `f54df2c9f302d89231b127deaef226e4f37cd0e1058c7db4f3ed218f3c886698` |
| Current aggregate SHA-256 | `f54df2c9f302d89231b127deaef226e4f37cd0e1058c7db4f3ed218f3c886698` |

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
