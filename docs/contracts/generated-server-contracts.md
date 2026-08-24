# Generated Server Contracts

Wave 2 adds a generated Go view of the OpenAPI HTTP surface. The generated file
is intentionally small and deterministic:

- `ServerInterface` lists one handler method for every OpenAPI operation;
- `Operation` records method, path, operation ID, tags, Laravel route metadata,
  authentication mode, trust path, tenant scope, and known consumers;
- `Operations` preserves the OpenAPI document order for future router, smoke,
  and differential harnesses.

The generator does not implement handlers and does not change traffic ownership.
Laravel remains the runtime owner for compatibility routes until a later ticket
adds fixtures, differential checks, tenant isolation tests, rollout gates, and
native Go handlers.

## Regeneration

Regenerate the Go contract after editing `api/openapi.yaml`:

```sh
python3 scripts/generate_openapi_server_contracts.py \
  --openapi api/openapi.yaml \
  --output internal/openapicontract/generated_routes.go
```

`make openapi-check` and `make verify` fail when the generated Go contract is not
in sync with `api/openapi.yaml`.

## Deployment and Rollback

This is contract and compile-time metadata only. It does not change runtime
routing, database access, or tenant data. Rollback is a Git revert of the
generator, generated Go file, tests, and documentation.
