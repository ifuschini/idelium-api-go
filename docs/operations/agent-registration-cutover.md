# Agent registration cutover

Issue #155 moves agent inventory, CLI registration, and browser status updates to the Go API.

All reads and writes are scoped by the active customer. Registration is idempotent on
`(idCostumer, agentId)`, preserves approval status on refresh, and stores no credentials.
The browser routes require `agents.read` or `agents.manage`; CLI registration uses the
tenant resolved from the API key. Roll back by routing these three endpoints to Laravel;
the shared table remains compatible and no dual writes are enabled.

Verification: `make openapi-check`, Docker Go tests, and the Laravel `AgentRegistrationTest`
must pass before enabling the gateway cutover.
