# Step CRUD cutover

Issue #134 is implemented for browser step list, detail, create, update, delete, and
ordering operations. Every query includes the active customer and project boundary;
mutations validate ownership before writing. Laravel remains the rollback owner until
gateway cutover, with no dual writes.
