# Laravel Schema Freeze

This generated report freezes the Laravel migration tree after the reviewed
Go baseline has been produced. It prevents accidental Laravel schema drift
during the final handover wave.

## Status

| Field | Value |
| --- | --- |
| Freeze status | `frozen` |
| Baseline ID | `go-baseline-2026-08-25` |
| Expected migrations | 68 |
| Current migrations | 68 |
| Expected aggregate SHA-256 | `fce30c061633192969cd77a1336cf8946a835859e33fe66cdfcee053e17f6fae` |
| Current aggregate SHA-256 | `fce30c061633192969cd77a1336cf8946a835859e33fe66cdfcee053e17f6fae` |

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
