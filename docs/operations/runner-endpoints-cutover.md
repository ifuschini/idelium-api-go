# Runner endpoint cutover

Issue #156 migrates the runner-only claim, heartbeat, and worker-update endpoints.
Each endpoint resolves the tenant from the API-key middleware, requires the short-lived
run token where applicable, validates bounded identifiers and lease values, and updates
worker state under a tenant/project/run row lock. Laravel remains the rollback owner until
the gateway switch is enabled; no dual writes are introduced.
