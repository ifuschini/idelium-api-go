-- Disposable, tenant-scoped fixtures for consumer smoke tests.
-- The API key and password below are synthetic values for local smoke only.
INSERT INTO costumers (id, costumer, description, apiKey, apiKeyExpiresAt)
VALUES (9001, 'fixture-customer-smoke', 'Disposable smoke tenant',
        'fixture-cli-key-9001', CURRENT_TIMESTAMP + INTERVAL 1 DAY);

INSERT INTO users (id, name, email, password, role, idCostumer, status)
VALUES (9001, 'Smoke Browser User', 'fixture-smoke@example.invalid',
        '$2y$12$vjGbzVIBFwfjJG21xaVEUua1ATd1RcszC5XikxqtWa8qDzCvv3NWy',
        2, 9001, 'active');

INSERT INTO projects (id, name, description, idCostumer)
VALUES (9001, 'fixture-project-smoke', 'Disposable smoke project', 9001);

INSERT INTO environments (id, name, code, description, config, idProject, idCostumer)
VALUES (9001, 'fixture-environment-smoke', 'fixture-environment-smoke', 'Disposable smoke environment', '{}', 9001, 9001);

INSERT INTO plugins (id, name, code, description, config, idProject, idCostumer)
VALUES (9001, 'fixture-plugin-smoke', 'fixture-plugin-smoke', 'Disposable smoke plugin', '{}', 9001, 9001);

INSERT INTO steps (id, name, description, config, idProject, idCostumer, `order`, sort)
VALUES (9001, 'fixture-step-smoke', 'Disposable smoke step', '{}', 9001, 9001, 1, 1);

INSERT INTO tests (id, name, description, config, idProject, idCostumer)
VALUES (9001, 'fixture-test-smoke', 'Disposable smoke test', '{"steps":[9001]}', 9001, 9001);

INSERT INTO test_cycles (id, name, description, config, idProject, idCostumer)
VALUES (9001, 'fixture-cycle-smoke', 'Disposable smoke cycle', '{"tests":[9001]}', 9001, 9001);

INSERT INTO parallel_run_schedules (
  id, idCostumer, idProject, testCycleId, idempotencyKey, status,
  requestedConcurrency, workerStates, resultSummary, metadata, scheduledAt
)
VALUES (
  9001, 9001, 9001, 9001, 'fixture-parallel-run-smoke', 'scheduled',
  1, '{"worker-smoke":{"status":"queued"}}', '{}',
  '{"fixture":"fixture-parallel-run-smoke"}', CURRENT_TIMESTAMP
);

INSERT INTO agent_registrations (id, idCostumer, agentId, capabilities)
VALUES (9001, 9001, 'fixture-agent-smoke', '["selenium"]');
