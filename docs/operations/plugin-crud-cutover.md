# Plugin CRUD cutover

Issue #133 migrates browser plugin list, detail, create, update, and delete routes.
Manifests must be non-empty JSON objects; all reads and writes enforce active tenant
and project ownership. Laravel remains the rollback owner until gateway cutover and
no dual writes are enabled.
