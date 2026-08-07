# Continuous Integration Gates

Every push and pull request runs three bounded, independent GitHub Actions jobs:

| Job | Required evidence |
| --- | --- |
| `quality` | Locked modules, formatting, vet, unit tests, race tests, contract tests, and all binary builds. |
| `integration` | MySQL package tests against the pinned MariaDB service and isolated test schema. |
| `image` | Reproducible container build and verification of numeric non-root runtime user `65532:65532`. |

Actions are pinned to full commit identifiers, the Go patch release is fixed,
and the MariaDB service uses an immutable image digest. Each job has a 15-minute
timeout. Superseded runs on the same ref are cancelled to avoid wasting runner
capacity while preserving the latest evidence.

`tests/test_ci_contract.py` prevents required gates, pins, timeouts, and the
non-root image assertion from being removed silently. A failed gate blocks the
change; it must not be bypassed by moving verification to an untracked local
command.

This CI change does not move traffic, access tenant-owned data, modify a schema,
or change a public HTTP contract. Laravel-Go differential and negative
cross-tenant tests are therefore not applicable. Rollback is a revert of the
dedicated workflow commit, although weakening a required gate needs explicit
engineering approval.
