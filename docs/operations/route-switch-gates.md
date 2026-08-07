# Release, Route-Switch, and Rollback Gates

## Purpose

This policy controls how an Idelium API route or transaction-owning aggregate
moves from Laravel to Go. The machine-readable source is
[`route-switch-gates.json`](route-switch-gates.json), validated during
`make verify`. Passing tests makes a change eligible for approval; it does not
authorize production traffic by itself.

## Change record

Every route switch must identify:

- route IDs and aggregate from the migration ownership matrix;
- source and target release revisions and immutable container digests;
- read-route or mutation-aggregate approval profile;
- links to every required evidence artifact;
- baseline and candidate metric windows in the same environment;
- named approvers, timestamps, planned observation window, and rollback owner;
- gateway configuration revision before and after the switch.

Do not put credentials, authorization headers, cookies, session identifiers,
raw customer payloads, or tenant identifiers in the record. Use safe correlation
surrogates and redacted dashboards.

## Required evidence

Before a release candidate can receive traffic, it requires:

1. an OpenAPI request and response contract;
2. a sanitized Laravel fixture;
3. a passing Laravel-Go differential test;
4. authorization and negative cross-tenant tests;
5. Web, CLI, runner, or Docker consumer smoke tests as applicable;
6. redacted logs, metrics, and traces with an operational dashboard;
7. load, timeout, and resource baselines;
8. a successful rollback rehearsal in the target deployment topology.

Database-backed changes additionally require empty-schema and supported-upgrade
migration verification against MySQL. A mutation switch cannot be approved for
one route when another writer remains in the same transaction-owning aggregate.

## Approvals

| Change | Required roles |
| --- | --- |
| Read route | API maintainer and operations on-call |
| Mutation aggregate | API maintainer, database owner, operations on-call, and security reviewer |

The author cannot satisfy a required independent review role unless the incident
process explicitly records an emergency exception. Approvals apply to the named
revision and route set; code, schema, gateway, or evidence changes invalidate
them.

## Progressive switch

Go is first deployed dark. After readiness and rollback rehearsal, eligible
traffic progresses through 5%, 25%, 50%, and 100% stages. Each stage must satisfy
its minimum observation period and all thresholds before promotion. Mutation
traffic must be partitioned at the gateway so a given aggregate has only one
writer; percentage-based duplication or application-level dual writes are
prohibited.

The 100% stage retains a 24-hour observation window before Laravel fallback
retirement can be considered. Removing fallback, dropping backward-compatible
schema, or extracting a database requires a separate ADR and release.

## Stop and rollback conditions

Immediately stop promotion and roll back on any:

- cross-tenant access or credential exposure;
- lost, duplicate, or mismatched write side effect;
- consumer smoke failure;
- breach of the documented 5xx, authorization-denial, or p95 latency threshold;
- inability to attribute or redact operational diagnostics;
- Go readiness failure or loss of the verified Laravel fallback.

Rollback freezes promotion, switches the gateway owner to Laravel, verifies
Laravel readiness and consumer smoke tests, preserves redacted diagnostics, and
opens an incident review. It does not perform reverse application replay or a
database restore because dual writes are forbidden and coexistence schema changes
must be backward compatible.

An API maintainer or operations on-call may trigger emergency rollback without
waiting for the full approval group. Forward rollout always requires the normal
approval profile again.

## Current deployment impact

This baseline policy moves no traffic, changes no route ownership, performs no
writes, and changes no schema. Laravel remains the effective and fallback owner.
OpenAPI and Laravel-Go differential execution are not applicable to this
governance-only ticket; they become mandatory evidence for an actual route
switch. Rollback of the policy itself is a Git revert.
