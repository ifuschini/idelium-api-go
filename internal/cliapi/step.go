package cliapi

import (
	"context"
	"time"
)

// Step is the Laravel-compatible CLI step read contract.
type Step struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Config      string     `json:"config"`
	IDProject   int64      `json:"idProject"`
	Order       int64      `json:"order"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	IDCostumer  int64      `json:"idCostumer"`
}

// StepRepository reads tenant-scoped CLI step configuration.
type StepRepository interface {
	GetStep(ctx context.Context, customerID int64, stepID int64) (Step, error)
}
