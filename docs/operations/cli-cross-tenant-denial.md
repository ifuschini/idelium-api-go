# CLI Cross-Tenant Denial Coverage

Wave 4 CLI configuration reads are tenant-owned. A resource that belongs to a
different customer must be indistinguishable from a missing resource and must
return the legacy `404 {"message":"Invalid id"}` behavior for single-resource
reads.

## Coverage matrix

| Resource | Lookup key | Tenant predicate | Expected denial |
| --- | --- | --- | --- |
| Test cycle | `test_cycles.id` | `test_cycles.idCostumer` | `cliapi.ErrNotFound` |
| Test | `tests.id` | `tests.idCostumer` | `cliapi.ErrNotFound` |
| Step | `steps.id` | `steps.idCostumer` | `cliapi.ErrNotFound` |
| Plugin | `plugins.id` | `plugins.idCostumer` | `cliapi.ErrNotFound` |
| Environment | `environments.id` | `environments.idCostumer` | `cliapi.ErrNotFound` |

List reads also include the project id and customer id in the same SQL query.
They return `[]` for valid projects with no tenant-owned rows, preserving the
Laravel CLI behavior while avoiding tenant enumeration.

## Verification

`tests/test_cli_cross_tenant_denial_coverage.py` checks that every CLI graph
resource has explicit MySQL integration coverage. The integration tests create
same-schema foreign rows and assert that a request from another customer cannot
load them.

This coverage complements the handler and router tests that assert the public
legacy `Invalid id` response shape.
