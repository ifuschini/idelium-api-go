# CLI test-cycle reads

Wave 4 moves `GET /api/ideliumcl/testcycle/{idTestCycle}` to Go ownership as
the first CLI configuration graph read. The route is intentionally read-only and
uses the legacy customer API-key middleware documented in
[legacy-cli-api-key-authentication.md](legacy-cli-api-key-authentication.md).

## Compatibility contract

The Go handler preserves the Laravel CLI contract:

- authentication uses the `Idelium-Key` header;
- successful reads return the legacy `test_cycles` row shape with `id`,
  `name`, `description`, `config`, `idProject`, timestamps, and `idCostumer`;
- malformed, missing, and cross-tenant identifiers return HTTP `404` with
  `{"message":"Invalid id"}`;
- missing or invalid customer API keys return HTTP `401` with
  `{"message":"Invalid key"}`;
- repository failures return the stable Go error envelope without leaking
  credentials, SQL details, or tenant data.

## Tenant isolation

The MySQL query constrains `test_cycles.id` and `test_cycles.idCostumer` in the
same read. A valid customer key cannot distinguish another customer's test cycle
from a missing test cycle. This preserves the Laravel behavior tested by
Idelium CLI tenant-isolation coverage.

## Route ownership

The rollout override marks only `GET|HEAD
/api/ideliumcl/testcycle/{idTestCycle}` as Go-owned. Mutating
`/api/ideliumcl/testcycle` routes remain Laravel-owned until the later result
write and authoring waves.

## Deployment and rollback

Deployments route this read to the Go API through the strangler gateway after
the Wave 4 smoke checks pass. Rollback is route-level: remove the rollout
override and route the read back to Laravel. No schema change and no dual write
are introduced by this slice.
