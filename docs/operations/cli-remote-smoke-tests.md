# CLI Remote Smoke Tests

The CLI remote smoke runner exercises the Go-owned configuration-read routes
that the Idelium CLI needs before it can execute a remote test cycle. The runner
is opt-in because it requires a live API, tenant-owned fixture identifiers, and
a legacy customer API key supplied by CI or the operator.

## Command

```bash
IDELIUM_CLI_SMOKE_GO_BASE_URL=https://go-api.example.internal \
IDELIUM_CLI_SMOKE_API_KEY=<redacted> \
IDELIUM_CLI_SMOKE_ID_TEST_CYCLE=7 \
IDELIUM_CLI_SMOKE_ID_TEST=9 \
IDELIUM_CLI_SMOKE_ID_STEP=12 \
IDELIUM_CLI_SMOKE_ID_PROJECT=3 \
IDELIUM_CLI_SMOKE_ID_PLUGIN=14 \
IDELIUM_CLI_SMOKE_ID_ENVIRONMENT=16 \
python3 scripts/run_cli_smoke.py --owner go --mode configuration-read
```

`IDELIUM_CLI_SMOKE_IDELIUM_KEY` is accepted as a compatibility alias for
`IDELIUM_CLI_SMOKE_API_KEY`.

## Coverage

The runner reads
[`docs/contracts/cli-smoke-targets.json`](../contracts/cli-smoke-targets.json)
and selects routes with:

- owner `go`;
- execution mode `configuration-read`;
- legacy API-key authentication.

It currently validates the complete Wave 4 Go-owned remote configuration graph:

- test-cycle read;
- test read;
- step read;
- plugin list and single-plugin read;
- environment list and single-environment read.

The runner sends only `GET` requests for the default Wave 4 mode and does not
perform result-reporting writes. It reads a small response prefix to verify that
the endpoint responds without persisting payload content.

## Safety

- Runtime credentials are read only from environment variables.
- API keys are never printed.
- Missing inputs report variable names only.
- Remote connection errors include route identifiers and sanitized reachability
  diagnostics without headers or credential values.

## Rollback

The runner does not move route traffic. If a route is rolled back to Laravel,
regenerate `docs/contracts/cli-smoke-targets.json`; the smoke runner will then
target the owner recorded by the generated plan.
