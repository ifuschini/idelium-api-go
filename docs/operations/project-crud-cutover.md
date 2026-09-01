# Project CRUD cutover

Issue #131 provides tenant-scoped browser project list, detail, create, update, and
transactional cascade delete. Deletion locks the project and removes dependent test,
step, plugin, environment, and performed-result rows only for the active customer.
Laravel remains the rollback owner until gateway cutover; no dual writes are used.
