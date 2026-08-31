# Asset impact and version review cutover

Issue #148 moves the five asset-impact and asset-version review routes to the Go
service. The shared Laravel tables remain unchanged, so route ownership can be
switched without a data migration or dual writes.

## Security and consistency

- Every lookup includes the active tenant and requested project. Missing,
  cross-tenant, and cross-project identifiers return the same not-found result.
- Reads require `resources.read`; review transitions require
  `resources.manage`.
- Impact traversal accepts only the documented asset types and interprets
  numeric references in bounded test and test-cycle configuration records.
- Version snapshots are immutable. Detail responses apply sensitive-key
  redaction before returning the stored snapshot.
- Review events and their redacted audit events are appended in one database
  transaction. The current version row and latest review state are locked while
  validating the transition.
- Authors cannot approve their own versions. Valid state changes are `draft` to
  `in_review`, `in_review` to `approved` or `deprecated`, and `approved` to
  `deprecated`.

## Deployment

1. Apply the existing Laravel migration baseline containing `asset_versions`
   and `asset_version_review_events`.
2. Deploy the pinned Go image and verify readiness.
3. Run the Go MySQL integration suite and the Laravel `AssetImpactTest` and
   `AssetVersioningTest` compatibility tests against isolated databases.
4. Route the five documented paths to Go and monitor correlation identifiers,
   not-found rates, validation failures, and database transaction errors.

Only one runtime may own review-event writes at a time. Existing immutable
version and review records are shared and require no replay.

## Compatibility evidence

An automated request-for-request differential is not used for the complete
workflow because review transitions append immutable state and replaying the
same transition against a shared fixture changes the valid next state. Instead,
the Laravel `AssetImpactTest` and `AssetVersioningTest` suites are executed in an
isolated PHP container as the reference contract, while Go handler and MySQL
integration tests assert the same status codes, envelopes, diff structure,
transition rules, authorization, audit effects, and tenant/project denials.

## Rollback

Route the five paths back to Laravel and stop Go review writes. Do not delete or
reverse review events: both runtimes use the same append-only schema and Laravel
can continue from the latest committed state. A database restore is unnecessary
unless an independent database incident occurred.
