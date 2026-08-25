# Laravel upgrade verification

Issue [#172](https://github.com/ifuschini/idelium-api-go/issues/172) adds a
Wave 10 gate for databases upgraded from the last Laravel-owned release.

The verifier compares the Laravel-compatible `migrations` table against the
reviewed Go baseline manifest. It does not read application tables, tenant rows,
payload values, credentials, headers, cookies, or session identifiers.

## Command

Run the verifier against a non-production copy of the last Laravel-owned
release:

```sh
go run ./cmd/migrate \
  --verify-laravel-upgrade \
  --from-laravel-release <release-or-commit>
```

The source release is operator-supplied because the migration gate must be tied
to the exact release artifact being rehearsed. The command reads database
configuration from `IDELIUM_DB_*` or compatible `DB_*` variables.

## Result states

| Status | Meaning | Exit code |
| --- | --- | ---: |
| `ready` | All reviewed Laravel migration markers are present and Go baseline application is enabled. | 0 |
| `blocked` | All reviewed markers are present, but Go baseline application is still disabled by Wave 10 policy. | 2 |
| `failed` | One or more reviewed baseline markers are missing. | 2 |
| `review-required` | The database contains migration markers outside the reviewed baseline. | 2 |

Database connection or inspection failures return exit code `1` with redacted
diagnostics.

## Current expected result

For a database that already contains every reviewed Laravel migration marker,
the current baseline returns `blocked` because Go baseline application remains
disabled until route cutover and rollback rehearsal pass.

That result is intentional: it proves the source schema is compatible while
preventing accidental Go-owned migration application during coexistence.

## Rollback

The verifier is read-only. Rollback is a Git revert of the verifier and
documentation. It does not change route ownership, create tables, update rows,
or require a database restore.
