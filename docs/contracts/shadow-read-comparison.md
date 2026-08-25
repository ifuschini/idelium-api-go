# Shadow Read Comparison

Go-owned safe reads use an offline shadow-read comparison before live gateway
mirroring. The comparison loads the sanitized Laravel golden fixture and the
sanitized Go golden fixture for every Go-owned safe read, then applies the same
path-only-safe comparator used by the differential harness. The plan started
with Wave 3 platform catalog reads and now also covers the Wave 6 managed
platform read used by launcher configuration.

Run:

```sh
python3 scripts/compare_shadow_reads.py
```

The command emits JSON shaped for CI annotations:

- `passed`: global boolean;
- `routes`: one record per compared route;
- `differences`: path-only diagnostics when a route differs.

The comparison is intentionally read-only and offline. It performs no HTTP
requests, writes no database rows, and never needs browser cookies, API keys, or
tenant credentials. Live request mirroring can be added later only after the
same redaction and path-only diagnostic rules are preserved.

Rollback is a Git revert of this plan or a route-level fallback to Laravel in
[`gateway-route-ownership.json`](gateway-route-ownership.json).
