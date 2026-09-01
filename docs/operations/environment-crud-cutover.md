# Environment CRUD cutover

Issue #132 migrates browser environment list, detail, create, update, and delete routes.
All operations enforce active tenant and project ownership. Inline secret, password,
token, and cookie keys are rejected on writes and recursively redacted on reads.
Laravel remains the rollback owner until gateway cutover; no dual writes are enabled.
