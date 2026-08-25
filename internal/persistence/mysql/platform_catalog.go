package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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

// ListOperatingSystems returns operating systems for one platform type using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListOperatingSystems(ctx context.Context, query platforms.OperatingSystemQuery) (platforms.OperatingSystemPage, error) {
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

	where, args := operatingSystemWhereClause(query)
	total, err := repository.countRows(ctx, "os", where, args)
	if err != nil {
		return platforms.OperatingSystemPage{}, err
	}

	sqlQuery := "SELECT id, name, type, created_at, updated_at FROM os" + where + " ORDER BY " + sortColumn + " " + query.Direction
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
		return platforms.OperatingSystemPage{}, fmt.Errorf("query platform operating systems: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.OperatingSystemItem, 0)
	for rows.Next() {
		var item platforms.OperatingSystemItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &createdAt, &updatedAt); err != nil {
			return platforms.OperatingSystemPage{}, fmt.Errorf("scan platform operating-system row: %w", err)
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
		return platforms.OperatingSystemPage{}, fmt.Errorf("read platform operating-system rows: %w", err)
	}

	page := platforms.OperatingSystemPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.OperatingSystemPageMeta{
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

// ListOperatingSystemVersions returns OS versions for one operating system using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListOperatingSystemVersions(ctx context.Context, query platforms.OperatingSystemVersionQuery) (platforms.OperatingSystemVersionPage, error) {
	sortColumns := map[string]string{
		"id":         "id",
		"version":    "version",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "version"
		query.Sort = "version"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := operatingSystemVersionWhereClause(query)
	total, err := repository.countRows(ctx, "version_os", where, args)
	if err != nil {
		return platforms.OperatingSystemVersionPage{}, err
	}

	sqlQuery := "SELECT id, version, idOs, created_at, updated_at FROM version_os" + where + " ORDER BY " + sortColumn + " " + query.Direction
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
		return platforms.OperatingSystemVersionPage{}, fmt.Errorf("query platform operating-system versions: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.OperatingSystemVersionItem, 0)
	for rows.Next() {
		var item platforms.OperatingSystemVersionItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Version, &item.IDOs, &createdAt, &updatedAt); err != nil {
			return platforms.OperatingSystemVersionPage{}, fmt.Errorf("scan platform operating-system version row: %w", err)
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
		return platforms.OperatingSystemVersionPage{}, fmt.Errorf("read platform operating-system version rows: %w", err)
	}

	page := platforms.OperatingSystemVersionPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.OperatingSystemVersionPageMeta{
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

// ListBrowsers returns browsers for one operating system using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListBrowsers(ctx context.Context, query platforms.BrowserQuery) (platforms.BrowserPage, error) {
	sortColumns := map[string]string{
		"id":         "id",
		"name":       "name",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "name"
		query.Sort = "name"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := browserWhereClause(query)
	total, err := repository.countRows(ctx, "browsers", where, args)
	if err != nil {
		return platforms.BrowserPage{}, err
	}

	sqlQuery := "SELECT id, name, idOs, created_at, updated_at FROM browsers" + where + " ORDER BY " + sortColumn + " " + query.Direction
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
		return platforms.BrowserPage{}, fmt.Errorf("query platform browsers: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.BrowserItem, 0)
	for rows.Next() {
		var item platforms.BrowserItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Name, &item.IDOs, &createdAt, &updatedAt); err != nil {
			return platforms.BrowserPage{}, fmt.Errorf("scan platform browser row: %w", err)
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
		return platforms.BrowserPage{}, fmt.Errorf("read platform browser rows: %w", err)
	}

	page := platforms.BrowserPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.BrowserPageMeta{
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

// ListBrowserVersions returns browser versions for one browser using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListBrowserVersions(ctx context.Context, query platforms.BrowserVersionQuery) (platforms.BrowserVersionPage, error) {
	sortColumns := map[string]string{
		"id":         "id",
		"version":    "version",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "version"
		query.Sort = "version"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := browserVersionWhereClause(query)
	total, err := repository.countRows(ctx, "version_browsers", where, args)
	if err != nil {
		return platforms.BrowserVersionPage{}, err
	}

	sqlQuery := "SELECT id, version, idBrowser, created_at, updated_at FROM version_browsers" + where + " ORDER BY " + sortColumn + " " + query.Direction
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
		return platforms.BrowserVersionPage{}, fmt.Errorf("query platform browser versions: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.BrowserVersionItem, 0)
	for rows.Next() {
		var item platforms.BrowserVersionItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.Version, &item.IDBrowser, &createdAt, &updatedAt); err != nil {
			return platforms.BrowserVersionPage{}, fmt.Errorf("scan platform browser version row: %w", err)
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
		return platforms.BrowserVersionPage{}, fmt.Errorf("read platform browser version rows: %w", err)
	}

	page := platforms.BrowserVersionPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.BrowserVersionPageMeta{
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

// ListManagedPlatforms returns configured execution platforms using the Laravel-compatible grid contract.
func (repository *PlatformCatalogRepository) ListManagedPlatforms(ctx context.Context, query platforms.ManagedPlatformQuery) (platforms.ManagedPlatformPage, error) {
	sortColumns := map[string]string{
		"hostname":           "hostname",
		"brandDescription":   "brandDescription",
		"osDescription":      "osDescription",
		"browserDescription": "browserDescription",
		"status":             "status",
		"created_at":         "created_at",
		"updated_at":         "updated_at",
	}
	sortColumn, ok := sortColumns[query.Sort]
	if !ok {
		sortColumn = "osDescription"
		query.Sort = "osDescription"
	}
	if query.Direction != "desc" {
		query.Direction = "asc"
	}

	where, args := managedPlatformWhereClause(query)
	total, err := repository.countRows(ctx, "platforms", where, args)
	if err != nil {
		return platforms.ManagedPlatformPage{}, err
	}

	sqlQuery := `SELECT id, type, hostname, location, os, osversion, brand, browser,
			brandDescription, osDescription, browserDescription, status, created_at, updated_at
		FROM platforms` + where + " ORDER BY " + sortColumn + " " + query.Direction + ", id ASC"
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

	items, err := repository.queryManagedPlatforms(ctx, sqlQuery, args...)
	if err != nil {
		return platforms.ManagedPlatformPage{}, err
	}

	page := platforms.ManagedPlatformPage{Data: items}
	if query.IsPaged() {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = platforms.ManagedPlatformPageMeta{
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

// ListLaunchTargets returns safe Web launcher target candidates.
func (repository *PlatformCatalogRepository) ListLaunchTargets(ctx context.Context, projectID int64) ([]platforms.LaunchTargetItem, error) {
	now := time.Now().UTC()
	targets := []platforms.LaunchTargetItem{
		{
			ID:           "platform-pool",
			Name:         "Platform pool",
			Type:         "platform-pool",
			Runtime:      "selenium",
			Capabilities: []string{"browserOverride", "parallel"},
			Capacity:     platforms.LaunchTargetCapacity{Available: 1, Max: 1, Queued: 0},
			Health:       "healthy",
			LastHealthAt: &now,
			Region:       "project",
		},
	}
	_ = projectID

	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT p.id, p.type, COALESCE(t.name, ''), p.hostname, p.location,
				p.osDescription, p.browserDescription, p.status, p.updated_at
		 FROM platforms p
		 LEFT JOIN types t ON t.id = p.type
		 ORDER BY p.osDescription ASC, p.id ASC
		 LIMIT 49`,
	)
	if err != nil {
		return nil, fmt.Errorf("query launcher targets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id                 int64
			typeID             int64
			typeName           string
			hostname           string
			location           int64
			osDescription      string
			browserDescription string
			status             int64
			updatedAt          sql.NullTime
		)
		if err := rows.Scan(&id, &typeID, &typeName, &hostname, &location, &osDescription, &browserDescription, &status, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan launcher target row: %w", err)
		}
		health := "disabled"
		available := 0
		if status == 1 {
			health = "healthy"
			available = 1
		}
		name := strings.TrimSpace(hostname)
		if name == "" {
			name = strings.TrimSpace(strings.Join([]string{osDescription, browserDescription}, " · "))
		}
		if name == "" {
			name = fmt.Sprintf("Platform %d", id)
		}
		runtime := "selenium"
		if strings.Contains(strings.ToLower(typeName), "mobile") {
			runtime = "appium"
		}
		platformID := id
		var lastHealthAt *time.Time
		if updatedAt.Valid {
			lastHealthAt = &updatedAt.Time
		}
		_ = typeID
		targets = append(targets, platforms.LaunchTargetItem{
			ID:           fmt.Sprintf("platform-%d", id),
			Name:         name,
			Type:         "platform",
			Runtime:      runtime,
			Capabilities: []string{"browserOverride"},
			Capacity:     platforms.LaunchTargetCapacity{Available: available, Max: 1, Queued: 0},
			Health:       health,
			LastHealthAt: lastHealthAt,
			Region:       fmt.Sprintf("%d", location),
			Hostname:     hostname,
			Browser:      browserDescription,
			IDPlatform:   &platformID,
			PlatformID:   &platformID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read launcher target rows: %w", err)
	}

	return targets, nil
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

func (repository *PlatformCatalogRepository) queryManagedPlatforms(ctx context.Context, query string, args ...any) ([]platforms.ManagedPlatformItem, error) {
	rows, err := repository.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query managed platforms: %w", err)
	}
	defer rows.Close()

	items := make([]platforms.ManagedPlatformItem, 0)
	for rows.Next() {
		var item platforms.ManagedPlatformItem
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Hostname,
			&item.Location,
			&item.OS,
			&item.OSVersion,
			&item.Brand,
			&item.Browser,
			&item.BrandDescription,
			&item.OSDescription,
			&item.BrowserDescription,
			&item.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan managed platform row: %w", err)
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
		return nil, fmt.Errorf("read managed platform rows: %w", err)
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

func operatingSystemWhereClause(query platforms.OperatingSystemQuery) (string, []any) {
	where, args := gridWhereClause(query.Search, "name", query.FilterIDs)
	typeClause := "type = ?"
	if where == "" {
		return " WHERE " + typeClause, []any{query.TypeID}
	}
	return where + " AND " + typeClause, append(args, query.TypeID)
}

func operatingSystemVersionWhereClause(query platforms.OperatingSystemVersionQuery) (string, []any) {
	where, args := gridWhereClause(query.Search, "version", query.FilterIDs)
	osClause := "idOs = ?"
	if where == "" {
		return " WHERE " + osClause, []any{query.IDOs}
	}
	return where + " AND " + osClause, append(args, query.IDOs)
}

func browserWhereClause(query platforms.BrowserQuery) (string, []any) {
	where, args := gridWhereClause(query.Search, "name", query.FilterIDs)
	osClause := "idOs = ?"
	if where == "" {
		return " WHERE " + osClause, []any{query.IDOs}
	}
	return where + " AND " + osClause, append(args, query.IDOs)
}

func browserVersionWhereClause(query platforms.BrowserVersionQuery) (string, []any) {
	where, args := gridWhereClause(query.Search, "version", query.FilterIDs)
	browserClause := "idBrowser = ?"
	if where == "" {
		return " WHERE " + browserClause, []any{query.IDBrowser}
	}
	return where + " AND " + browserClause, append(args, query.IDBrowser)
}

func managedPlatformWhereClause(query platforms.ManagedPlatformQuery) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	clauses = append(clauses, "type = ?")
	args = append(args, query.TypeID)
	if query.Search != "" {
		clauses = append(clauses, "(hostname LIKE ? OR brandDescription LIKE ? OR osDescription LIKE ? OR browserDescription LIKE ?)")
		search := "%" + query.Search + "%"
		args = append(args, search, search, search, search)
	}
	if len(query.FilterIDs) > 0 {
		placeholders := make([]string, len(query.FilterIDs))
		for index, id := range query.FilterIDs {
			placeholders[index] = "?"
			args = append(args, id)
		}
		clauses = append(clauses, "id IN ("+strings.Join(placeholders, ",")+")")
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
