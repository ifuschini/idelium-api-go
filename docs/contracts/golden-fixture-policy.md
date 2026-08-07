# Sanitized Golden Fixture Policy

## Purpose

Golden fixtures preserve the externally observable Laravel contract while a
route moves to Go. They are test evidence, not production captures: every
committed value must be synthetic, deterministic, bounded, and safe to publish.
The machine-readable format is defined by
[`golden-fixture.schema.json`](golden-fixture.schema.json), and committed
fixtures are checked by `scripts/validate_golden_fixtures.py` during
`make verify`.

## Required record

Each fixture must contain:

- a stable fixture identifier and format version;
- the source runtime, immutable source revision, capture timestamp, and route
  inventory digest;
- HTTP method, templated route path, and canonical trust path;
- synthetic tenant and actor context when the route is tenant-owned;
- sanitized request and response status, headers, query, and body;
- normalized nondeterministic fields and a manifest of removed sensitive data;
- observable database or event side effects, represented only by synthetic
  identifiers and contract-relevant fields.

The committed example is
[`testdata/golden/platform-status.fixture.json`](../../testdata/golden/platform-status.fixture.json).

## Sanitization requirements

Before a fixture is written to disk:

1. Replace customer, tenant, user, project, test, and run data with deliberately
   synthetic values. Tenant and actor identifiers must start with `fixture-` and
   declare `synthetic: true`.
2. Remove authorization, proxy authorization, cookies, session identifiers,
   API keys, CSRF tokens, passwords, private keys, environment secrets, and raw
   test payload secrets. Do not retain these values in redacted form inside
   request or response headers; list the removed location in `redactions`.
3. Use reserved domains such as `example.test` or `example.invalid` for
   synthetic identities. `idelium.org` is allowed only as a public product URL,
   never as a synthetic customer identity.
4. Normalize timestamps, UUIDs, correlation IDs, generated database IDs, and
   unordered collections. Record every normalization as a JSON path and rule.
5. Retain only fields needed to compare status codes, response shape, selected
   headers, authorization outcomes, ordering, pagination, and side effects.
6. Review the final diff as public data. A passing validator supplements human
   review and secret scanning; it does not replace either control.

The redaction marker is `[REDACTED]`. A marker documents absence and must not be
used as a request credential during replay.

## Limits

| Resource | Limit |
| --- | ---: |
| Fixture file | 256 KiB |
| Serialized request body | 64 KiB |
| Serialized response body | 64 KiB |
| Headers per request or response | 50 |
| Side-effect records | 100 |

Artifacts and large response bodies must be represented by a sanitized digest,
media type, and byte count in a later artifact fixture. They must not be embedded
in this format.

## Capture and update workflow

1. Select a route from `compatibility-backlog.json` and create synthetic tenant
   state in an isolated non-production database.
2. Capture Laravel behavior at an immutable commit and record its route inventory
   digest. Never capture from a production tenant.
3. Sanitize and normalize the record before it enters the repository.
4. Run `python3 scripts/validate_golden_fixtures.py testdata/golden` and
   `make verify`.
5. Review fixture changes together with the contract decision that caused them.
   Do not automatically accept new output merely to make a differential test
   pass.
6. Version breaking fixture-format changes with a new `fixtureVersion`; retain a
   reader for the previous supported version until its migration is complete.

Fixtures are immutable evidence for a specific source revision. Behavioral
changes create a new fixture or an explicitly reviewed replacement in the same
route migration commit.

## Compatibility, deployment, and rollback

This policy does not expose an HTTP operation, change a database schema, move
traffic, or alter Laravel behavior. Laravel remains the route owner. A
Laravel-versus-Go differential test is therefore not applicable to this policy
ticket; the policy is the prerequisite for the Wave 2 comparator.

Deployment consists of publishing the versioned policy and running its validator
in CI. Rollback is a Git revert. A rollback must never restore a fixture removed
because it contained sensitive or real tenant data; rotate any potentially
exposed credential and follow the repository security process instead.
