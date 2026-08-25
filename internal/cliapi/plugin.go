package cliapi

import (
	"context"
	"time"
)

// Plugin is the Laravel-compatible CLI plugin read contract.
type Plugin struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description string     `json:"description"`
	IDProject   int64      `json:"idProject"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	IDCostumer  int64      `json:"idCostumer"`
}

// PluginRepository reads tenant-scoped CLI plugin configuration.
type PluginRepository interface {
	ListPlugins(ctx context.Context, customerID int64, projectID int64) ([]Plugin, error)
	GetPlugin(ctx context.Context, customerID int64, pluginID int64) (Plugin, error)
}
