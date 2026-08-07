# Repository Foundation Contract

The repository foundation is an enforced delivery contract rather than a set
of optional starter files.

- `AGENTS.md` extends the shared Idelium engineering rules with Go API,
  OpenAPI-first, tenant-isolation, compatibility, and verification directives.
- `README.md` describes the current migration scope, endpoints, configuration,
  local verification, and Apache 2.0 licensing.
- `LICENSE` contains the Apache License 2.0 text and Idelium attribution.
- `Makefile` exposes consistent formatting, vet, unit, race, contract,
  integration, and build commands.
- `Dockerfile` pins the builder by immutable digest, verifies Go modules,
  produces a trimmed static binary, and runs it from `scratch` as numeric
  non-root user `65532:65532`.

`tests/test_repository_foundation.py` prevents accidental removal or weakening
of these properties. Docker and dependency version changes must be intentional,
reviewed, and accompanied by a passing verification run.

This ticket does not move a Laravel route, change persistence, access tenant
data, or affect runtime side effects. Differential, database migration, and
negative cross-tenant tests are therefore not applicable. Rollback consists of
reverting the dedicated repository-foundation commit.
