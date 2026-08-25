package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/idelium/idelium-api-go/internal/platforms"
)

// PlatformCatalogRepository reads legacy global platform catalog tables.
type PlatformCatalogRepository struct {
	database *sql.DB
}

// NewPlatformCatalogRepository creates a MySQL-backed catalog repository.
func NewPlatformCatalogRepository(database *sql.DB) *PlatformCatalogRepository {
	return &PlatformCatalogRepository{database: database}
}

// ListTypes returns platform target types ordered like the Laravel endpoint.
func (repository *PlatformCatalogRepository) ListTypes(ctx context.Context) ([]platforms.CatalogItem, error) {
	return repository.listCatalogItems(ctx, "SELECT id, name FROM types ORDER BY id ASC")
}

// ListStatuses returns platform statuses ordered like the Laravel endpoint.
func (repository *PlatformCatalogRepository) ListStatuses(ctx context.Context) ([]platforms.CatalogItem, error) {
	return repository.listCatalogItems(ctx, "SELECT id, name FROM statuses ORDER BY id ASC")
}

// ListLocations returns platform locations using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListLocations(ctx context.Context, query platforms.LocationQuery) (platforms.LocationPage, error) {
	sortColumns := map[string]string{
		"id":         "id",
		"name":       "name",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "id"
		query.Sort = "id"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := locationWhereClause(query)
	total, err := repository.countLocations(ctx, where, args)
	if err != nil {
		return platforms.LocationPage{}, err
	}

	sqlQuery := "SELECT id, name, created_at, updated_at FROM locations" + where + " ORDER BY " + sortColumn + " " + query.Direction
	if query.IsPaged() {
		pageSize := query.PageSize
		if pageSize == 0 {
			pageSize = 25
		}
		page := query.Page
		if page == 0 {
			page = 1
		}
		args = append(args, pageSize, (page-1)*pageSize)
		sqlQuery += " LIMIT ? OFFSET ?"
		query.Page = page
		query.PageSize = pageSize
	}

	rows, err := repository.database.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return platforms.LocationPage{}, fmt.Errorf("query platform locations: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.LocationItem, 0)
	for rows.Next() {
		var item platforms.LocationItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Name, &createdAt, &updatedAt); err != nil {
			return platforms.LocationPage{}, fmt.Errorf("scan platform location row: %w", err)
		}
		if createdAt.Valid {
			item.CreatedAt = &createdAt.Time
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return platforms.LocationPage{}, fmt.Errorf("read platform location rows: %w", err)
	}

	page := platforms.LocationPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.LocationPageMeta{
			Page:            query.Page,
			PageSize:        query.PageSize,
			Total:           total,
			LastPage:        lastPage,
			HasNextPage:     query.Page < lastPage,
			HasPreviousPage: query.Page > 1,
			Sort:            query.Sort,
			Direction:       query.Direction,
			Stale:           false,
			Partial:         false,
		}
	}

	return page, nil
}

// ListBrands returns device brands using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListBrands(ctx context.Context, query platforms.BrandQuery) (platforms.BrandPage, error) {
	sortColumns := map[string]string{
		"id":         "id",
		"brand":      "brand",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "id"
		query.Sort = "id"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := gridWhereClause(query.Search, "brand", query.FilterIDs)
	total, err := repository.countRows(ctx, "brand_devices", where, args)
	if err != nil {
		return platforms.BrandPage{}, err
	}

	sqlQuery := "SELECT id, brand, created_at, updated_at FROM brand_devices" + where + " ORDER BY " + sortColumn + " " + query.Direction
	if query.IsPaged() {
		pageSize := query.PageSize
		if pageSize == 0 {
			pageSize = 25
		}
		page := query.Page
		if page == 0 {
			page = 1
		}
		args = append(args, pageSize, (page-1)*pageSize)
		sqlQuery += " LIMIT ? OFFSET ?"
		query.Page = page
		query.PageSize = pageSize
	}

	rows, err := repository.database.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return platforms.BrandPage{}, fmt.Errorf("query platform brands: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.BrandItem, 0)
	for rows.Next() {
		var item platforms.BrandItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Brand, &createdAt, &updatedAt); err != nil {
			return platforms.BrandPage{}, fmt.Errorf("scan platform brand row: %w", err)
		}
		if createdAt.Valid {
			item.CreatedAt = &createdAt.Time
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return platforms.BrandPage{}, fmt.Errorf("read platform brand rows: %w", err)
	}

	page := platforms.BrandPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.BrandPageMeta{
			Page:            query.Page,
			PageSize:        query.PageSize,
			Total:           total,
			LastPage:        lastPage,
			HasNextPage:     query.Page < lastPage,
			HasPreviousPage: query.Page > 1,
			Sort:            query.Sort,
			Direction:       query.Direction,
			Stale:           false,
			Partial:         false,
		}
	}

	return page, nil
}

// ListModels returns device models for one brand using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListModels(ctx context.Context, query platforms.ModelQuery) (platforms.ModelPage, error) {
	sortColumns := map[string]string{
		"id":         "id",
		"model":      "model",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "model"
		query.Sort = "model"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := modelWhereClause(query)
	total, err := repository.countRows(ctx, "model_devices", where, args)
	if err != nil {
		return platforms.ModelPage{}, err
	}

	sqlQuery := "SELECT id, model, idBrand, created_at, updated_at FROM model_devices" + where + " ORDER BY " + sortColumn + " " + query.Direction
	if query.IsPaged() {
		pageSize := query.PageSize
		if pageSize == 0 {
			pageSize = 25
		}
		page := query.Page
		if page == 0 {
			page = 1
		}
		args = append(args, pageSize, (page-1)*pageSize)
		sqlQuery += " LIMIT ? OFFSET ?"
		query.Page = page
		query.PageSize = pageSize
	}

	rows, err := repository.database.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return platforms.ModelPage{}, fmt.Errorf("query platform models: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.ModelItem, 0)
	for rows.Next() {
		var item platforms.ModelItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Model, &item.IDBrand, &createdAt, &updatedAt); err != nil {
			return platforms.ModelPage{}, fmt.Errorf("scan platform model row: %w", err)
		}
		if createdAt.Valid {
			item.CreatedAt = &createdAt.Time
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return platforms.ModelPage{}, fmt.Errorf("read platform model rows: %w", err)
	}

	page := platforms.ModelPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.ModelPageMeta{
			Page:            query.Page,
			PageSize:        query.PageSize,
			Total:           total,
			LastPage:        lastPage,
			HasNextPage:     query.Page < lastPage,
			HasPreviousPage: query.Page > 1,
			Sort:            query.Sort,
			Direction:       query.Direction,
			Stale:           false,
			Partial:         false,
		}
	}

	return page, nil
}

func (repository *PlatformCatalogRepository) listCatalogItems(ctx context.Context, query string) ([]platforms.CatalogItem, error) {
	rows, err := repository.database.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query platform catalog: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.CatalogItem, 0)
	for rows.Next() {
		var item platforms.CatalogItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan platform catalog row: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read platform catalog rows: %w", err)
	}

	return items, nil
}

func (repository *PlatformCatalogRepository) countLocations(ctx context.Context, where string, args []any) (int64, error) {
	return repository.countRows(ctx, "locations", where, args)
}

func (repository *PlatformCatalogRepository) countRows(ctx context.Context, table string, where string, args []any) (int64, error) {
	var total int64
	if err := repository.database.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+where, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count platform catalog rows: %w", err)
	}
	return total, nil
}

func locationWhereClause(query platforms.LocationQuery) (string, []any) {
	return gridWhereClause(query.Search, "name", query.FilterIDs)
}

func gridWhereClause(search string, searchColumn string, filterIDs []int64) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if search != "" {
		clauses = append(clauses, searchColumn+" LIKE ?")
		args = append(args, "%"+search+"%")
	}
	if len(filterIDs) > 0 {
		placeholders := make([]string, len(filterIDs))
		for index, id := range filterIDs {
			placeholders[index] = "?"
			args = append(args, id)
		}
		clauses = append(clauses, "id IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func modelWhereClause(query platforms.ModelQuery) (string, []any) {
	where, args := gridWhereClause(query.Search, "model", query.FilterIDs)
	brandClause := "idBrand = ?"
	if where == "" {
		return " WHERE " + brandClause, []any{query.IDBrand}
	}
	return where + " AND " + brandClause, append(args, query.IDBrand)
}
