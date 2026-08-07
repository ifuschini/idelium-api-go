# Laravel Route Inventory Baseline

> This planning summary is complemented by the generated
> [complete route inventory](laravel-routes.md) and its
> [machine-readable contract](laravel-routes.json). Consumer evidence is tracked
> in the generated [route consumer map](consumer-route-map.md).

## Purpose

This inventory is the starting point for contract extraction from `idelium-api`.
It is not a declaration that the current Laravel behavior is automatically the
desired long-term contract. Each route must be classified, secured, documented,
and covered by consumer and differential tests before Go assumes ownership.

## Baseline metrics

The Laravel route file currently declares:

| Method declaration | Count |
| --- | ---: |
| `GET` | 63 |
| `POST` | 61 |
| `PUT` | 28 |
| `DELETE` | 6 |
| Resource declaration | 1 |

The project resource declaration expands to additional REST operations. The
route surface is backed by 46 controllers, 41 models, and 66 migrations.

## Trust paths

| Trust path | Current mechanism | Migration priority |
| --- | --- | --- |
| Browser | Sanctum session, cookie, CSRF, tenant context | Last |
| CLI | Customer or service-account Idelium key | Early |
| Runner | Run token and worker lease | Late/high concurrency |
| Workload | OIDC workload token exchange | Late/identity |
| SSO | OIDC/SAML callback and identity lifecycle | Last |

The generated inventory assigns every route to one canonical migration trust
path: `browser-session`, `api-key`, `run-token`, `internal-service`, or
`public-operational`. The more detailed authentication mode is retained beside
that classification so bootstrap and callback behavior is not obscured.

`public-operational` is an inventory classification, not a security approval.
It intentionally exposes the current public root, cache-clear route, and local
Ignition routes for explicit deployment hardening and compatibility decisions.

## Domain route groups

| Domain | Representative route prefixes | Proposed wave |
| --- | --- | ---: |
| Operations | `/health/*` | 1 |
| Platform catalog | `/api/admin/platforms/*` | 3 |
| CLI configuration | `/api/ideliumcl/testcycle`, `test`, `step`, `plugin`, `environment` | 4 |
| CLI results | `/api/ideliumcl/testcycle`, `test`, `step` mutations | 5 |
| Projects and authoring | `/api/admin/projects`, `environments`, `plugins`, `steps`, `tests`, `testcycles`, `importtest` | 6 |
| Result exploration | `/api/admin/*perfomed`, `/api/admin/result-exports` | 7 |
| Artifacts and governance | `/api/admin/projects/*/artifacts`, asset impact and versions | 7 |
| Grid and integrations | `/api/admin/grid`, `/api/admin/projects/*/integrations` | 7 |
| Parallel execution | `/api/admin/projects/*/parallel-runs`, `/api/ideliumrunner/*` | 8 |
| Agents | `/api/admin/agents`, `/api/ideliumcl/agents` | 8 |
| Browser identity | `/api/login`, `/api/logout`, `/api/sanctum`, profile, accounts, customers | 9 |
| Enterprise identity | `/api/sso`, MFA, SCIM, service accounts, workload identity | 9 |

## Contract record required per route

Before implementation, record:

- operation ID and current route name;
- request consumer and minimum supported consumer version;
- authentication mode and authorization capability;
- tenant ownership and cross-tenant response behavior;
- request fields, types, limits, and validation errors;
- response fields, optionality, ordering, pagination, filtering, and sorting;
- status codes and stable error codes;
- transaction, idempotency, audit, and background-job effects;
- sensitive data classification and redaction policy;
- Laravel fixture, Go fixture, and differential test identifier;
- route owner, rollout state, and rollback state.

## First reference slice

The first business slice will use platform catalog reads after the health and
contract foundation is green. Although these routes are structurally simple,
their existing role behavior must be reviewed before it is treated as a stable
contract. The current Laravel implementation can return a plain `"ok"` payload
for unauthorized roles, which should not be copied into Go without an explicit
compatibility and security decision.

## Inventory deployment and rollback

The generated inventory does not move traffic, change a database schema, or
alter Laravel behavior. Deployment therefore consists only of publishing the
versioned contract artifacts. Rollback is a normal Git revert of the inventory
commit. Any unexpected route delta must be investigated in Laravel before an
updated baseline is accepted; it must not be hidden by editing generated files.

Laravel-Go differential execution is not applicable to this inventory-only
ticket because no HTTP behavior changes. The exported contracts are the input
to differential coverage in later route migration tickets.
