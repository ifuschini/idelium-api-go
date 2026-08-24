# Golden Side-Effect Comparator

Issue [#98](https://github.com/ifuschini/idelium-api-go/issues/98) adds an
offline comparator for mutation fixtures. The safe-read HTTP comparator remains
responsible for `GET` and `HEAD` response equivalence; this comparator is
responsible for database side effects produced by `POST`, `PUT`, `PATCH`, and
`DELETE` routes.

## Contract

Mutation fixtures must declare `sideEffects` as a sanitized ordered list. Each
record should contain the table, operation, synthetic tenant identifier, stable
natural key, and the row projection required to prove Laravel-Go equivalence.
The comparator checks:

- mutation-only route methods;
- route method, path, trust path, and tenant ownership metadata;
- non-empty side-effect declarations;
- side-effect shape, order, field presence, JSON types, and values;
- fixture-declared normalizations under `$.sideEffects`.

Diagnostics are path-only and never echo row values, credentials, session
identifiers, cookies, authorization headers, or payload secrets.

## Normalization

Fixtures may normalize generated values such as database IDs, UUIDs, timestamps,
and correlation identifiers. Normalizations are explicit JSON-path declarations
under `normalizations`; missing or unsupported paths fail the comparison instead
of being ignored.

## Compatibility and rollback

This tool does not execute database writes and does not move traffic. It compares
already captured, sanitized Laravel and Go fixtures before any mutation route is
allowed to become Go-owned. Rollback is a normal revert of the comparator or the
route-owner change that required the side-effect fixture.
