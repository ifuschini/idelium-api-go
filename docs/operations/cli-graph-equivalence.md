# CLI Graph Equivalence

Wave 4 migrates the remote configuration graph that the Idelium CLI downloads
before executing a test cycle. Route-level golden fixtures prove that each
endpoint preserves its Laravel contract; graph equivalence proves that the same
cycle still resolves to the same connected resources when Go owns the reads.

## Graph scope

The graph fixture covers one synthetic tenant-owned cycle:

- one test cycle with ordered test identifiers;
- one test with ordered step identifiers;
- one executable step;
- one project-scoped plugin;
- one project-scoped environment.

The comparison validates:

- Laravel and Go graph payloads are identical after source-runtime metadata is
  normalized;
- every cycle-referenced test exists;
- every test-referenced step exists;
- every test, step, plugin, and environment belongs to the same customer and
  project as the graph context.

## Fixtures

- `testdata/golden/cli-graph-cycle.fixture.json`
- `testdata/golden/cli-graph-cycle-go.fixture.json`

The fixtures are synthetic and contain no credentials, request headers, payload
secrets, cookies, or session identifiers.

## Command

```bash
python3 scripts/compare_cli_graph.py \
  --expected testdata/golden/cli-graph-cycle.fixture.json \
  --actual testdata/golden/cli-graph-cycle-go.fixture.json
```

The comparison also runs through `make verify` via the Python test suite.

## Rollback

This evidence does not move traffic. If a graph read route is rolled back to
Laravel, keep the fixture pair and update only the route ownership and smoke
target plans. If the Laravel contract changes, capture a new sanitized Laravel
fixture before changing the Go fixture.
