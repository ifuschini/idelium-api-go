package platforms

import "context"

// CatalogItem is a global platform catalog row exposed by legacy Laravel routes.
type CatalogItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CatalogRepository reads global platform reference data.
type CatalogRepository interface {
	ListTypes(ctx context.Context) ([]CatalogItem, error)
	ListStatuses(ctx context.Context) ([]CatalogItem, error)
}
