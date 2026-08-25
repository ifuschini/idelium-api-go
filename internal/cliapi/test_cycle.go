package cliapi

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound hides missing and cross-tenant CLI resources behind the same response.
	ErrNotFound = errors.New("CLI resource not found")
)

// TestCycle is the Laravel-compatible CLI test-cycle read contract.
type TestCycle struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Config      string     `json:"config"`
	IDProject   int64      `json:"idProject"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	IDCostumer   int64      `json:"idCostumer"`
}

// TestCycleRepository reads tenant-scoped CLI test-cycle configuration.
type TestCycleRepository interface {
	GetTestCycle(ctx context.Context, customerID int64, testCycleID int64) (TestCycle, error)
}
