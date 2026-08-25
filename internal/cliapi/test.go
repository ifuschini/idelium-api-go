package cliapi

import (
	"context"
	"time"
)

// Test is the Laravel-compatible CLI test read contract.
type Test struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Config      string     `json:"config"`
	IDProject   int64      `json:"idProject"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	IDCostumer   int64      `json:"idCostumer"`
}

// TestRepository reads tenant-scoped CLI test configuration.
type TestRepository interface {
	GetTest(ctx context.Context, customerID int64, testID int64) (Test, error)
}
