# Golden HTTP Comparator

The safe-read comparator checks that a sanitized Go fixture preserves the
externally observable Laravel behavior for a route that has no side effects. It
is the first Wave 2 differential building block and intentionally supports only
`GET` and `HEAD` fixtures.

The comparator checks:

- route method, path, trust path, and tenant ownership metadata;
- HTTP status;
- comparable response headers such as `Content-Type`, `Cache-Control`, and
  `Location`;
- response body shape and values.

Before comparing, the comparator applies declared `normalizations` from both
fixtures. Supported normalization paths use the fixture JSON-path style already
defined by the golden fixture policy, for example:

- `$.response.body.createdAt` for timestamps;
- `$.response.body.id` or nested array paths such as `$.response.body[0].id`
  for generated identifiers or UUIDs;
- `$.response.body.correlationId` or `$.request.headers.X-Correlation-ID` for
  correlation identifiers.

Each declared normalization is applied to both fixtures with a stable marker. If
a declared path is missing from either fixture, the comparison fails with a
path-only diagnostic so stale or overly broad normalization rules cannot hide
real contract drift.

The comparator rejects mutation fixtures and any fixture that declares side
effects. Mutation and database comparison belongs to the later side-effect
comparator ticket.

## Safe Diagnostics

Comparison output reports only JSON paths and generic reasons. It does not print
request headers, authorization values, cookies, body values, tenant identifiers,
or payload content. This keeps failed CI diagnostics useful without turning
fixtures into a secret disclosure path.

## Usage

```sh
python3 scripts/compare_golden_http.py \
  --expected testdata/golden/platform-status.fixture.json \
  --actual testdata/golden/platform-status-go.fixture.json
```

The command exits with `0` when fixtures match and `1` when drift is detected.
The emitted JSON is suitable for CI annotations or future differential reports.

## Deployment and Rollback

This tool changes no HTTP routing, database schema, tenant data, or runtime side
effects. Laravel remains the route owner until later Wave 2 gates pass. Rollback
is a Git revert of the comparator, tests, and documentation.
