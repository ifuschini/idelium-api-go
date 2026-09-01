-- Disposable schema for the Go consumer-smoke profile only.
CREATE TABLE costumers (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  costumer VARCHAR(255) NOT NULL,
  description TEXT NULL,
  ideliumKey VARCHAR(255) NULL,
  licenseExpiredAt TIMESTAMP NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  password VARCHAR(255) NOT NULL,
  role INT NOT NULL DEFAULT 2,
  idCostumer BIGINT NOT NULL,
  disabledAt TIMESTAMP NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE projects (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  description TEXT NULL,
  idCostumer BIGINT NOT NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE test_cycles (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  description TEXT NULL,
  config JSON NOT NULL,
  idProject BIGINT NOT NULL,
  idCostumer BIGINT NOT NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE tests (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  description TEXT NULL,
  config JSON NOT NULL,
  idProject BIGINT NOT NULL,
  idCostumer BIGINT NOT NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE steps (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  description TEXT NULL,
  config JSON NOT NULL,
  idProject BIGINT NOT NULL,
  idCostumer BIGINT NOT NULL,
  sort INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE plugins (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  config JSON NULL,
  idProject BIGINT NOT NULL,
  idCostumer BIGINT NOT NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE environments (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  config JSON NULL,
  idProject BIGINT NOT NULL,
  idCostumer BIGINT NOT NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE parallel_run_schedules (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  idCostumer BIGINT NOT NULL,
  idProject BIGINT NOT NULL,
  testCycleId BIGINT NOT NULL,
  performedTestCycleId BIGINT NULL,
  idempotencyKey VARCHAR(160) NOT NULL,
  status VARCHAR(32) NOT NULL,
  requestedConcurrency INT NOT NULL,
  activeWorkers INT NOT NULL DEFAULT 0,
  totalWorkers INT NOT NULL DEFAULT 0,
  completedWorkers INT NOT NULL DEFAULT 0,
  failedWorkers INT NOT NULL DEFAULT 0,
  cancelledWorkers INT NOT NULL DEFAULT 0,
  aggregateStatus INT NULL,
  workerStates JSON NOT NULL,
  resultSummary JSON NOT NULL,
  metadata JSON NOT NULL,
  scheduledAt TIMESTAMP NULL,
  startedAt TIMESTAMP NULL,
  completedAt TIMESTAMP NULL,
  cancelledAt TIMESTAMP NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL,
  UNIQUE KEY parallel_run_idempotency (idCostumer, idProject, idempotencyKey)
);

CREATE TABLE run_tokens (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  idCostumer BIGINT NOT NULL,
  idProject BIGINT NOT NULL,
  parallelRunScheduleId BIGINT NOT NULL,
  agentId VARCHAR(255) NOT NULL,
  tokenId VARCHAR(255) NOT NULL UNIQUE,
  tokenHash VARCHAR(255) NOT NULL,
  expiresAt TIMESTAMP NOT NULL,
  usedAt TIMESTAMP NULL,
  revokedAt TIMESTAMP NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE agent_registrations (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  idCostumer BIGINT NOT NULL,
  agentId VARCHAR(255) NOT NULL,
  certificateHash VARCHAR(255) NULL,
  capabilities JSON NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL,
  UNIQUE KEY agent_registration (idCostumer, agentId)
);

CREATE TABLE go_browser_sessions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  sessionHash CHAR(64) NOT NULL UNIQUE,
  csrfHash CHAR(64) NOT NULL,
  userId BIGINT NOT NULL,
  tenantId BIGINT NOT NULL,
  activeTenantId BIGINT NOT NULL,
  expiresAt TIMESTAMP NOT NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL
);

CREATE TABLE audit_events (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  idCostumer BIGINT NULL,
  actorUserId BIGINT NULL,
  event VARCHAR(255) NOT NULL,
  aggregateType VARCHAR(255) NULL,
  aggregateId BIGINT NULL,
  beforeValues JSON NULL,
  afterValues JSON NULL,
  status VARCHAR(32) NOT NULL,
  correlationId VARCHAR(255) NULL,
  created_at TIMESTAMP NULL
);
