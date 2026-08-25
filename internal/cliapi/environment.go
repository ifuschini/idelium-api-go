package cliapi

import (
	"context"
	"time"
)

// Environment is the Laravel-compatible CLI environment read contract.
type Environment struct {
	ID          int64      `json:"id"`
	Code        string     `json:"code"`
	Description string     `json:"description"`
	Config      string     `json:"config"`
	IDProject   int64      `json:"idProject"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	IDCostumer  int64      `json:"idCostumer"`
}

// EnvironmentRepository reads tenant-scoped CLI environment configuration.
type EnvironmentRepository interface {
	ListEnvironments(ctx context.Context, customerID int64, projectID int64) ([]Environment, error)
	GetEnvironment(ctx context.Context, customerID int64, environmentID int64) (Environment, error)
}
