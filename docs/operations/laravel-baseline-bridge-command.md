# Laravel baseline bridge command

Issue [#170](https://github.com/ifuschini/idelium-api-go/issues/170) adds an
operator command that marks the reviewed Laravel migration baseline as already
applied in a Laravel-compatible `migrations` table.

The bridge is intended for cutover rehearsals where the target database already
contains the Laravel schema and the Go runtime must avoid replaying equivalent
baseline migrations.

## Dry-run

Always inspect the plan first:

```sh
go run ./cmd/migrate \
  --mark-laravel-baseline-applied \
  --confirm-baseline-id go-baseline-2026-08-25 \
  --batch 67
```

The output is JSON and includes:

- the reviewed baseline ID;
- the number of migration markers;
- the target Laravel `migrations` table;
- the batch value;
- the idempotent SQL statement template;
- the migration names that would be marked.

The dry-run does not connect to the database.

## Execute

Execution requires the same explicit confirmation plus `--execute`:

```sh
go run ./cmd/migrate \
  --mark-laravel-baseline-applied \
  --confirm-baseline-id go-baseline-2026-08-25 \
  --batch 67 \
  --execute
```

The command reads the standard `IDELIUM_DB_*` or compatible `DB_*`
configuration. Passwords must be provided through environment variables or a
secret file as documented in
[database-configuration.md](database-configuration.md).

Each marker insert is idempotent:

```sql
INSERT INTO migrations (migration, batch)
SELECT ?, ?
WHERE NOT EXISTS (
  SELECT 1 FROM migrations WHERE migration = ?
)
```

The command reports only aggregate counters:

- `applied`;
- `skipped`;
- `migration_count`;
- `batch`;
- `baseline_id`.

It does not print database credentials, DSNs, connection strings, tenant data,
or migration payload values.

## Validation and rollback

The command fails before opening a database connection when:

- `--confirm-baseline-id` is missing;
- `--confirm-baseline-id` does not match the reviewed embedded baseline;
- `--batch` is not greater than zero.

Rollback is operationally simple because the command only marks an existing
schema baseline as applied. If a cutover rehearsal must be reverted, restore the
previous database snapshot or delete only the inserted bridge markers after
confirming no Go-owned migration has depended on that batch. Laravel remains the
schema owner until Wave 10 exit criteria pass.
