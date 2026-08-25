# Empty install verification

Issue [#171](https://github.com/ifuschini/idelium-api-go/issues/171) adds a
Wave 10 verification gate for empty database installs using the reviewed Go
baseline migration metadata.

The current reviewed baseline intentionally keeps Go baseline application
disabled. Therefore an empty schema is expected to report `blocked` until the
remaining upgrade, staging cutover, Docker default, and rollback rehearsal gates
pass.

## Command

Run the verifier against a non-production database configured with the standard
`IDELIUM_DB_*` or compatible `DB_*` variables:

```sh
go run ./cmd/migrate --verify-empty-install
```

The command inspects only `information_schema.tables` for the configured schema.
It does not read tenant rows, payload values, credentials, headers, cookies, or
session identifiers.

## Result states

| Status | Meaning | Exit code |
| --- | --- | ---: |
| `ready` | The schema is empty and Go baseline application is enabled. | 0 |
| `blocked` | The schema is empty, but policy prevents Go baseline application. | 2 |
| `failed` | The target schema already contains application tables. | 2 |

Database connection or inspection failures return exit code `1` with redacted
diagnostics.

## Current expected result

For the current baseline, an empty schema returns:

```json
{
  "status": "blocked",
  "schema_empty": true,
  "blockers": [
    "Go baseline application is disabled until bridge, upgrade, route cutover, and rollback rehearsal gates pass"
  ]
}
```

This is intentional. It proves the target database is suitable for an empty
install rehearsal while preserving the Wave 10 rule that Laravel remains the
schema owner until all cutover gates pass.

## Rollback

The verifier performs no writes, creates no tables, and moves no route traffic.
Rollback is a Git revert of the verifier and documentation if the gate needs to
be redesigned.
