package mysql

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

type BrowserAuthRepository struct{ database *sql.DB }

func NewBrowserAuthRepository(database *sql.DB) *BrowserAuthRepository {
	return &BrowserAuthRepository{database: database}
}

func (r *BrowserAuthRepository) ListRoles(_ *http.Request, actor browserauth.User) ([]browserauth.Role, bool, error) {
	if actor.Role > 2 {
		return nil, true, nil
	}
	query := `SELECT id, name, created_at, updated_at FROM roles`
	args := []any{}
	if actor.Role == 2 {
		query += ` WHERE id > ?`
		args = append(args, 1)
	}
	query += ` ORDER BY id ASC`
	rows, err := r.database.Query(query, args...)
	if err != nil {
		return nil, false, safeDatabaseFailure("list browser roles", err)
	}
	defer rows.Close()
	roles := []browserauth.Role{}
	for rows.Next() {
		var role browserauth.Role
		if err := rows.Scan(&role.ID, &role.Name, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, false, safeDatabaseFailure("scan browser roles", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, false, safeDatabaseFailure("read browser roles", err)
	}
	return roles, false, nil
}

func (r *BrowserAuthRepository) Profile(request *http.Request, actor browserauth.User) (browserauth.Profile, error) {
	var profile browserauth.Profile
	err := r.database.QueryRowContext(request.Context(), `SELECT users.email, users.name, costumers.costumer, roles.name
		FROM users
		JOIN costumers ON users.idCostumer = costumers.id
		JOIN roles ON users.role = roles.id
		WHERE users.id = ? AND users.idCostumer = ?
		LIMIT 1`, actor.ID, actor.TenantID).Scan(&profile.Email, &profile.Name, &profile.CompanyName, &profile.RoleName)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.Profile{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.Profile{}, safeDatabaseFailure("load browser profile", err)
	}
	return profile, nil
}

func (r *BrowserAuthRepository) UpdateProfilePassword(request *http.Request, actor browserauth.User, password string) (browserauth.Profile, error) {
	hash, err := browserauth.HashPasswordForRepository(password)
	if err != nil {
		return browserauth.Profile{}, err
	}
	result, err := r.database.ExecContext(request.Context(), `UPDATE users SET password = ?, updated_at = ? WHERE id = ? AND idCostumer = ?`, hash, time.Now().UTC(), actor.ID, actor.TenantID)
	if err != nil {
		return browserauth.Profile{}, safeDatabaseFailure("update browser profile password", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return browserauth.Profile{}, safeDatabaseFailure("count browser profile password update", err)
	}
	if affected == 0 {
		return browserauth.Profile{}, browserauth.ErrNotFound
	}
	return r.Profile(request, actor)
}

func (r *BrowserAuthRepository) ListAccounts(request *http.Request, actor browserauth.User, query browserauth.AccountQuery) (browserauth.AccountPage, error) {
	where, args := accountScope(actor)
	if query.Search != "" {
		where += ` AND (users.email LIKE ? OR users.name LIKE ? OR costumers.costumer LIKE ? OR roles.name LIKE ?)`
		search := "%" + boundedSearch(query.Search) + "%"
		args = append(args, search, search, search, search)
	}
	total, err := countAccounts(request.Context(), r.database, where, args)
	if err != nil {
		return browserauth.AccountPage{}, err
	}
	sort := accountSortColumn(query.Sort)
	direction := "ASC"
	if query.Direction == "desc" {
		direction = "DESC"
	}
	sqlQuery := `SELECT users.id, users.email, users.name, users.role, users.idCostumer, costumers.costumer, roles.name
		FROM users JOIN costumers ON users.idCostumer = costumers.id JOIN roles ON users.role = roles.id ` + where + ` ORDER BY ` + sort + ` ` + direction
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.AccountPage{}, safeDatabaseFailure("list browser accounts", err)
	}
	defer rows.Close()
	accounts := []browserauth.Account{}
	for rows.Next() {
		var account browserauth.Account
		if err := rows.Scan(&account.ID, &account.Email, &account.Name, &account.Role, &account.IDCostumer, &account.Costumer, &account.RoleName); err != nil {
			return browserauth.AccountPage{}, safeDatabaseFailure("scan browser account", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return browserauth.AccountPage{}, safeDatabaseFailure("read browser accounts", err)
	}
	page := browserauth.AccountPage{Data: accounts}
	if query.Paged {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = browserauth.PageMeta{Page: query.Page, PageSize: query.PageSize, Total: total, LastPage: lastPage, HasNextPage: query.Page < lastPage, HasPreviousPage: query.Page > 1, Sort: query.Sort, Direction: query.Direction, Stale: false, Partial: false}
	}
	return page, nil
}

func (r *BrowserAuthRepository) CreateAccount(request *http.Request, actor browserauth.User, account browserauth.AccountCreate) error {
	if !canManageAccount(actor, account.Role, account.IDCostumer) {
		return browserauth.ErrForbidden
	}
	if ok, err := existsByID(request.Context(), r.database, "roles", account.Role); err != nil || !ok {
		if err != nil {
			return err
		}
		return browserauth.ErrForbidden
	}
	if ok, err := existsByID(request.Context(), r.database, "costumers", account.IDCostumer); err != nil || !ok {
		if err != nil {
			return err
		}
		return browserauth.ErrForbidden
	}
	var duplicate int
	err := r.database.QueryRowContext(request.Context(), `SELECT 1 FROM users WHERE email = ? LIMIT 1`, account.Email).Scan(&duplicate)
	if err == nil {
		return browserauth.ErrForbidden
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return safeDatabaseFailure("check browser account email", err)
	}
	hash, err := browserauth.HashPasswordForRepository(account.Password)
	if err != nil {
		return err
	}
	_, err = r.database.ExecContext(request.Context(), `INSERT INTO users (name, password, email, role, idCostumer, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)`, account.Name, hash, account.Email, account.Role, account.IDCostumer, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return safeDatabaseFailure("create browser account", err)
	}
	return nil
}

func (r *BrowserAuthRepository) UpdateAccount(request *http.Request, actor browserauth.User, account browserauth.AccountUpdate) error {
	target, err := r.accountTarget(request, actor, account.ID)
	if err != nil {
		return err
	}
	if !canManageAccount(actor, target.Role, target.IDCostumer) {
		return browserauth.ErrForbidden
	}
	hash, err := browserauth.HashPasswordForRepository(account.Password)
	if err != nil {
		return err
	}
	result, err := r.database.ExecContext(request.Context(), `UPDATE users SET name = ?, password = ?, updated_at = ? WHERE id = ?`, account.Name, hash, time.Now().UTC(), account.ID)
	if err != nil {
		return safeDatabaseFailure("update browser account", err)
	}
	return requireAffected(result)
}

func (r *BrowserAuthRepository) DeleteAccount(request *http.Request, actor browserauth.User, accountID int64) error {
	target, err := r.accountTarget(request, actor, accountID)
	if err != nil {
		return err
	}
	if !canManageAccount(actor, target.Role, target.IDCostumer) {
		return browserauth.ErrForbidden
	}
	result, err := r.database.ExecContext(request.Context(), `DELETE FROM users WHERE id = ?`, accountID)
	if err != nil {
		return safeDatabaseFailure("delete browser account", err)
	}
	return requireAffected(result)
}

func (r *BrowserAuthRepository) ListAdminCustomers(request *http.Request, query browserauth.CustomerQuery) (browserauth.CustomerPage, error) {
	where := `WHERE 1 = 1`
	args := []any{}
	if query.Search != "" {
		where += ` AND (costumer LIKE ? OR description LIKE ?)`
		search := "%" + boundedSearch(query.Search) + "%"
		args = append(args, search, search)
	}
	var total int64
	if err := r.database.QueryRowContext(request.Context(), `SELECT COUNT(*) FROM costumers `+where, args...).Scan(&total); err != nil {
		return browserauth.CustomerPage{}, safeDatabaseFailure("count browser customers", err)
	}
	sort := customerSortColumn(query.Sort)
	direction := "ASC"
	if query.Direction == "desc" {
		direction = "DESC"
	}
	sqlQuery := `SELECT id, costumer, description, licenseExpiration, created_at, updated_at FROM costumers ` + where + ` ORDER BY ` + sort + ` ` + direction
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.CustomerPage{}, safeDatabaseFailure("list browser customers", err)
	}
	defer rows.Close()
	customers := []browserauth.Customer{}
	for rows.Next() {
		var customer browserauth.Customer
		if err := rows.Scan(&customer.ID, &customer.Costumer, &customer.Description, &customer.LicenseExpiration, &customer.CreatedAt, &customer.UpdatedAt); err != nil {
			return browserauth.CustomerPage{}, safeDatabaseFailure("scan browser customer", err)
		}
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return browserauth.CustomerPage{}, safeDatabaseFailure("read browser customers", err)
	}
	page := browserauth.CustomerPage{Data: customers}
	if query.Paged {
		lastPage := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))
		if lastPage == 0 {
			lastPage = 1
		}
		page.Meta = browserauth.PageMeta{Page: query.Page, PageSize: query.PageSize, Total: total, LastPage: lastPage, HasNextPage: query.Page < lastPage, HasPreviousPage: query.Page > 1, Sort: query.Sort, Direction: query.Direction, Stale: false, Partial: false}
	}
	return page, nil
}

func (r *BrowserAuthRepository) CreateCustomer(request *http.Request, customer browserauth.CustomerCreate) error {
	apiKey, err := randomAPIKey()
	if err != nil {
		return err
	}
	_, err = r.database.ExecContext(request.Context(), `INSERT INTO costumers
		(costumer, description, licenseExpiration, apiKey, apiKeyCreatedAt, logo, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '[]', ?, ?)`,
		strings.ToUpper(customer.Costumer),
		strings.ToUpper(customer.Description),
		customer.Now.AddDate(0, 0, 365),
		apiKey,
		customer.Now,
		customer.Now,
		customer.Now,
	)
	if err != nil {
		return safeDatabaseFailure("create browser customer", err)
	}
	return nil
}

func (r *BrowserAuthRepository) UpdateCustomer(request *http.Request, customer browserauth.CustomerUpdate) error {
	result, err := r.database.ExecContext(request.Context(), `UPDATE costumers SET costumer = ?, description = ?, updated_at = ? WHERE id = ?`, strings.ToUpper(customer.Costumer), strings.ToUpper(customer.Description), time.Now().UTC(), customer.ID)
	if err != nil {
		return safeDatabaseFailure("update browser customer", err)
	}
	return requireAffected(result)
}

func (r *BrowserAuthRepository) DeleteCustomer(request *http.Request, customerID int64) error {
	result, err := r.database.ExecContext(request.Context(), `DELETE FROM costumers WHERE id = ?`, customerID)
	if err != nil {
		return safeDatabaseFailure("delete browser customer", err)
	}
	return requireAffected(result)
}

func (r *BrowserAuthRepository) ListTestCycles(request *http.Request, actor browserauth.User, query browserauth.ResourceQuery) (browserauth.TestCyclePage, error) {
	if err := r.ensureProject(request.Context(), actor.ActiveTenant(), query.ProjectID); err != nil {
		return browserauth.TestCyclePage{}, err
	}
	where, args := resourceWhere(actor.ActiveTenant(), query.ProjectID, query)
	total, err := countRows(request.Context(), r.database, "test_cycles", where, args)
	if err != nil {
		return browserauth.TestCyclePage{}, err
	}
	sort := resourceSortColumn(query.Sort)
	direction := resourceDirection(query.Direction)
	sqlQuery := `SELECT id, name, description, created_at, updated_at FROM test_cycles ` + where + ` ORDER BY ` + sort + ` ` + direction
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.TestCyclePage{}, safeDatabaseFailure("list browser test cycles", err)
	}
	defer rows.Close()
	items := []browserauth.TestCycle{}
	for rows.Next() {
		var item browserauth.TestCycle
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return browserauth.TestCyclePage{}, safeDatabaseFailure("scan browser test cycle", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return browserauth.TestCyclePage{}, safeDatabaseFailure("read browser test cycles", err)
	}
	return browserauth.TestCyclePage{Data: items, Meta: pageMeta(query.Page, query.PageSize, total, query.Sort, query.Direction)}, nil
}

func (r *BrowserAuthRepository) CreateTestCycle(request *http.Request, actor browserauth.User, input browserauth.TestCycleCreate) error {
	ctx := request.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), input.IDProject); err != nil {
		return err
	}
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start browser test cycle create", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `INSERT INTO test_cycles (name, description, config, idProject, idCostumer, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, input.Name, input.Description, input.Config, input.IDProject, actor.ActiveTenant(), now, now)
	if err != nil {
		return safeDatabaseFailure("create browser test cycle", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return safeDatabaseFailure("read created browser test cycle id", err)
	}
	if err := recordAssetVersion(ctx, tx, actor, "test_cycle", id, input.IDProject, "asset.created", map[string]any{"id": id, "name": input.Name, "description": input.Description, "config": input.Config, "idProject": input.IDProject, "idCostumer": actor.ActiveTenant()}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *BrowserAuthRepository) GetTestCycle(request *http.Request, actor browserauth.User, projectID int64, cycleID int64) (browserauth.TestCycleDetail, error) {
	var cycle browserauth.TestCycleDetail
	err := r.database.QueryRowContext(request.Context(), `SELECT test_cycles.id, test_cycles.name, test_cycles.description, test_cycles.config, test_cycles.idProject
		FROM test_cycles
		JOIN projects ON projects.id = test_cycles.idProject AND projects.idCostumer = test_cycles.idCostumer
		WHERE test_cycles.id = ? AND test_cycles.idProject = ? AND test_cycles.idCostumer = ? LIMIT 1`, cycleID, projectID, actor.ActiveTenant()).Scan(&cycle.ID, &cycle.Name, &cycle.Description, &cycle.Config, &cycle.IDProject)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.TestCycleDetail{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.TestCycleDetail{}, safeDatabaseFailure("load browser test cycle", err)
	}
	return cycle, nil
}

func (r *BrowserAuthRepository) UpdateTestCycle(request *http.Request, actor browserauth.User, input browserauth.TestCycleUpdate) error {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start browser test cycle update", err)
	}
	defer tx.Rollback()
	var name string
	err = tx.QueryRowContext(ctx, `SELECT name FROM test_cycles WHERE id = ? AND idProject = ? AND idCostumer = ? LIMIT 1`, input.ID, input.IDProject, actor.ActiveTenant()).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ErrNotFound
	}
	if err != nil {
		return safeDatabaseFailure("load browser test cycle for update", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE test_cycles SET description = ?, config = ?, updated_at = ? WHERE id = ? AND idProject = ? AND idCostumer = ?`, input.Description, input.Config, time.Now().UTC(), input.ID, input.IDProject, actor.ActiveTenant())
	if err != nil {
		return safeDatabaseFailure("update browser test cycle", err)
	}
	if err := requireAffected(result); err != nil {
		return err
	}
	if err := recordAssetVersion(ctx, tx, actor, "test_cycle", input.ID, input.IDProject, "asset.updated", map[string]any{"id": input.ID, "name": name, "description": input.Description, "config": input.Config, "idProject": input.IDProject, "idCostumer": actor.ActiveTenant()}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *BrowserAuthRepository) ReorderSteps(request *http.Request, actor browserauth.User, input browserauth.StepReorder) error {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start browser step reorder", err)
	}
	defer tx.Rollback()
	if err := ensureProjectTx(ctx, tx, actor.ActiveTenant(), input.IDProject); err != nil {
		return err
	}
	for position, step := range input.Order {
		result, err := tx.ExecContext(ctx, `UPDATE steps SET `+"`order`"+` = ?, updated_at = ? WHERE id = ? AND idProject = ? AND idCostumer = ?`, input.Offset+position, time.Now().UTC(), step.ID, input.IDProject, actor.ActiveTenant())
		if err != nil {
			return safeDatabaseFailure("reorder browser step", err)
		}
		if err := requireAffected(result); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *BrowserAuthRepository) ListStepsForReorder(request *http.Request, actor browserauth.User, query browserauth.ResourceQuery) (browserauth.StepPage, error) {
	if err := r.ensureProject(request.Context(), actor.ActiveTenant(), query.ProjectID); err != nil {
		return browserauth.StepPage{}, err
	}
	where, args := resourceWhere(actor.ActiveTenant(), query.ProjectID, query)
	total, err := countRows(request.Context(), r.database, "steps", where, args)
	if err != nil {
		return browserauth.StepPage{}, err
	}
	sqlQuery := `SELECT id, name, description, ` + "`order`" + ` FROM steps ` + where + ` ORDER BY ` + resourceSortColumn(query.Sort) + ` ` + resourceDirection(query.Direction)
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.StepPage{}, safeDatabaseFailure("list browser steps after reorder", err)
	}
	defer rows.Close()
	items := []browserauth.StepListItem{}
	for rows.Next() {
		var item browserauth.StepListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Order); err != nil {
			return browserauth.StepPage{}, safeDatabaseFailure("scan browser step after reorder", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return browserauth.StepPage{}, safeDatabaseFailure("read browser steps after reorder", err)
	}
	return browserauth.StepPage{Data: items, Meta: pageMeta(query.Page, query.PageSize, total, query.Sort, query.Direction)}, nil
}

func (r *BrowserAuthRepository) ListTests(request *http.Request, actor browserauth.User, query browserauth.ResourceQuery) (browserauth.TestPage, error) {
	if err := r.ensureProject(request.Context(), actor.ActiveTenant(), query.ProjectID); err != nil {
		return browserauth.TestPage{}, err
	}
	where, args := resourceWhere(actor.ActiveTenant(), query.ProjectID, query)
	total, err := countRows(request.Context(), r.database, "tests", where, args)
	if err != nil {
		return browserauth.TestPage{}, err
	}
	sqlQuery := `SELECT id, name, description, created_at, updated_at FROM tests ` + where + ` ORDER BY ` + resourceSortColumn(query.Sort) + ` ` + resourceDirection(query.Direction)
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.TestPage{}, safeDatabaseFailure("list browser tests", err)
	}
	defer rows.Close()
	items := []browserauth.Test{}
	for rows.Next() {
		var item browserauth.Test
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return browserauth.TestPage{}, safeDatabaseFailure("scan browser test", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return browserauth.TestPage{}, safeDatabaseFailure("read browser tests", err)
	}
	return browserauth.TestPage{Data: items, Meta: pageMeta(query.Page, query.PageSize, total, query.Sort, query.Direction)}, nil
}

func (r *BrowserAuthRepository) CreateTest(request *http.Request, actor browserauth.User, input browserauth.TestCreate) error {
	ctx := request.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), input.IDProject); err != nil {
		return err
	}
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start browser test create", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `INSERT INTO tests (name, description, config, idProject, idCostumer, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, input.Name, input.Description, input.Config, input.IDProject, actor.ActiveTenant(), now, now)
	if err != nil {
		return safeDatabaseFailure("create browser test", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return safeDatabaseFailure("read created browser test id", err)
	}
	if err := recordAssetVersion(ctx, tx, actor, "test", id, input.IDProject, "asset.created", map[string]any{"id": id, "name": input.Name, "description": input.Description, "config": input.Config, "idProject": input.IDProject, "idCostumer": actor.ActiveTenant()}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *BrowserAuthRepository) GetTest(request *http.Request, actor browserauth.User, projectID int64, testID int64) (browserauth.TestDetail, error) {
	var test browserauth.TestDetail
	err := r.database.QueryRowContext(request.Context(), `SELECT tests.id, tests.name, tests.description, tests.config, tests.idProject
		FROM tests
		JOIN projects ON projects.id = tests.idProject AND projects.idCostumer = tests.idCostumer
		WHERE tests.id = ? AND tests.idProject = ? AND tests.idCostumer = ? LIMIT 1`, testID, projectID, actor.ActiveTenant()).Scan(&test.ID, &test.Name, &test.Description, &test.Config, &test.IDProject)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.TestDetail{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.TestDetail{}, safeDatabaseFailure("load browser test", err)
	}
	return test, nil
}

func (r *BrowserAuthRepository) UpdateTest(request *http.Request, actor browserauth.User, input browserauth.TestUpdate) error {
	ctx := request.Context()
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start browser test update", err)
	}
	defer tx.Rollback()
	var name string
	var description string
	err = tx.QueryRowContext(ctx, `SELECT name, description FROM tests WHERE id = ? AND idProject = ? AND idCostumer = ? LIMIT 1`, input.ID, input.IDProject, actor.ActiveTenant()).Scan(&name, &description)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ErrNotFound
	}
	if err != nil {
		return safeDatabaseFailure("load browser test for update", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE tests SET config = ?, updated_at = ? WHERE id = ? AND idProject = ? AND idCostumer = ?`, input.Config, time.Now().UTC(), input.ID, input.IDProject, actor.ActiveTenant())
	if err != nil {
		return safeDatabaseFailure("update browser test", err)
	}
	if err := requireAffected(result); err != nil {
		return err
	}
	if err := recordAssetVersion(ctx, tx, actor, "test", input.ID, input.IDProject, "asset.updated", map[string]any{"id": input.ID, "name": name, "description": description, "config": input.Config, "idProject": input.IDProject, "idCostumer": actor.ActiveTenant()}); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *BrowserAuthRepository) ImportTest(request *http.Request, actor browserauth.User, input browserauth.TestImport) error {
	ctx := request.Context()
	var imported []map[string]any
	if err := json.Unmarshal([]byte(input.Import), &imported); err != nil || len(imported) == 0 {
		return browserauth.ErrNotFound
	}
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start browser test import", err)
	}
	defer tx.Rollback()
	if err := ensureProjectTx(ctx, tx, actor.ActiveTenant(), input.IDProject); err != nil {
		return err
	}
	now := time.Now().UTC()
	importedTestConfig := make([]map[string]any, 0, len(imported))
	for _, importedStep := range imported {
		stepName, _ := importedStep["name"].(string)
		trimmedName := strings.TrimSpace(stepName)
		config, err := json.Marshal(importedStep)
		if err != nil {
			return fmt.Errorf("encode browser imported step config: %w", err)
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO steps (name, description, config, idProject, idCostumer, `+"`order`"+`, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, strings.ReplaceAll(trimmedName, " ", "_"), trimmedName, string(config), input.IDProject, actor.ActiveTenant(), 9999999, now, now)
		if err != nil {
			return safeDatabaseFailure("create browser imported step", err)
		}
		stepID, err := result.LastInsertId()
		if err != nil {
			return safeDatabaseFailure("read created browser imported step id", err)
		}
		importedTestConfig = append(importedTestConfig, map[string]any{"id": stepID, "name": strings.ReplaceAll(trimmedName, " ", "_"), "description": trimmedName})
	}
	testConfig, err := json.Marshal(importedTestConfig)
	if err != nil {
		return fmt.Errorf("encode browser imported test config: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO tests (name, description, config, idProject, idCostumer, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, input.Name, input.Description, string(testConfig), input.IDProject, actor.ActiveTenant(), now, now)
	if err != nil {
		return safeDatabaseFailure("create browser imported test", err)
	}
	return tx.Commit()
}

func (r *BrowserAuthRepository) ListPerformedCycles(request *http.Request, actor browserauth.User, query browserauth.ResultQuery) (browserauth.PerformedCyclePage, error) {
	where, args := resultWhere("testCycleId", actor.ActiveTenant(), query)
	total, err := countRows(request.Context(), r.database, "performed_test_cycles", where, args)
	if err != nil {
		return browserauth.PerformedCyclePage{}, err
	}
	sqlQuery := `SELECT id, testCycleId, date, status, updated_at, created_at FROM performed_test_cycles ` + where + ` ORDER BY ` + resultSortColumn(query.Sort, "date") + ` ` + resourceDirection(query.Direction)
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PerPage, (query.Page-1)*query.PerPage)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.PerformedCyclePage{}, safeDatabaseFailure("list browser performed cycles", err)
	}
	defer rows.Close()
	items := []browserauth.PerformedCycle{}
	for rows.Next() {
		var item browserauth.PerformedCycle
		if err := rows.Scan(&item.ID, &item.TestCycleID, &item.Date, &item.Status, &item.UpdatedAt, &item.CreatedAt); err != nil {
			return browserauth.PerformedCyclePage{}, safeDatabaseFailure("scan browser performed cycle", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return browserauth.PerformedCyclePage{}, safeDatabaseFailure("read browser performed cycles", err)
	}
	return browserauth.PerformedCyclePage{Data: items, Meta: resultMeta(query.Page, query.PerPage, total, query.Sort, query.Direction)}, nil
}

func (r *BrowserAuthRepository) ListPerformedTests(request *http.Request, actor browserauth.User, query browserauth.ResultQuery) (browserauth.PerformedTestPage, error) {
	where, args := resultWhere("testCycleDoneId", actor.ActiveTenant(), query)
	total, err := countRows(request.Context(), r.database, "performed_tests", where, args)
	if err != nil {
		return browserauth.PerformedTestPage{}, err
	}
	sqlQuery := `SELECT id, testCycleDoneId, testId, status, postmanData, name, updated_at, created_at FROM performed_tests ` + where + ` ORDER BY ` + resultSortColumn(query.Sort, "id") + ` ` + resourceDirection(query.Direction)
	if query.Paged {
		sqlQuery += ` LIMIT ? OFFSET ?`
		args = append(args, query.PerPage, (query.Page-1)*query.PerPage)
	}
	rows, err := r.database.QueryContext(request.Context(), sqlQuery, args...)
	if err != nil {
		return browserauth.PerformedTestPage{}, safeDatabaseFailure("list browser performed tests", err)
	}
	defer rows.Close()
	items := []browserauth.PerformedTest{}
	for rows.Next() {
		var item browserauth.PerformedTest
		var postmanData sql.NullString
		if err := rows.Scan(&item.ID, &item.TestCycleDoneID, &item.TestID, &item.Status, &postmanData, &item.Name, &item.UpdatedAt, &item.CreatedAt); err != nil {
			return browserauth.PerformedTestPage{}, safeDatabaseFailure("scan browser performed test", err)
		}
		if postmanData.Valid {
			redacted := redactResultJSONString(postmanData.String)
			item.PostmanData = &redacted
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return browserauth.PerformedTestPage{}, safeDatabaseFailure("read browser performed tests", err)
	}
	return browserauth.PerformedTestPage{Data: items, Meta: resultMeta(query.Page, query.PerPage, total, query.Sort, query.Direction)}, nil
}

func (r *BrowserAuthRepository) ListPerformedSteps(request *http.Request, actor browserauth.User, performedTestID int64) ([]browserauth.PerformedStep, error) {
	rows, err := r.database.QueryContext(request.Context(), `SELECT id, testCycleDoneId, testDoneId, name, status, screenshots, data, type, updated_at, created_at
		FROM performed_steps
		WHERE testDoneId = ? AND idCostumer = ?
		ORDER BY id ASC`, performedTestID, actor.ActiveTenant())
	if err != nil {
		return nil, safeDatabaseFailure("list browser performed steps", err)
	}
	defer rows.Close()
	items := []browserauth.PerformedStep{}
	for rows.Next() {
		var item browserauth.PerformedStep
		if err := rows.Scan(&item.ID, &item.TestCycleDoneID, &item.TestDoneID, &item.Name, &item.Status, &item.Screenshots, &item.Data, &item.Type, &item.UpdatedAt, &item.CreatedAt); err != nil {
			return nil, safeDatabaseFailure("scan browser performed step", err)
		}
		item.Screenshots = redactResultJSONString(item.Screenshots)
		item.Data = redactResultJSONString(item.Data)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser performed steps", err)
	}
	return items, nil
}

func (r *BrowserAuthRepository) CreateResultExport(request *http.Request, actor browserauth.User, input browserauth.ResultExportCreate) (browserauth.ResultExportDescriptor, error) {
	ctx := request.Context()
	run, err := r.performedCycle(ctx, actor.ActiveTenant(), input.PerformedTestCycleID)
	if err != nil {
		return browserauth.ResultExportDescriptor{}, err
	}
	payload, err := r.resultExportPayload(ctx, actor.ActiveTenant(), run, input.Format)
	if err != nil {
		return browserauth.ResultExportDescriptor{}, err
	}
	expiresAt := input.Now.Add(24 * time.Hour)
	filename := resultExportFilename(run.ID, input.Format)
	contentType := resultExportContentType(input.Format)
	result, err := r.database.ExecContext(ctx, `INSERT INTO result_exports (idCostumer, performedTestCycleId, format, status, filename, contentType, payload, errorMessage, expiresAt, created_at, updated_at) VALUES (?, ?, ?, 'completed', ?, ?, ?, NULL, ?, ?, ?)`, actor.ActiveTenant(), run.ID, input.Format, filename, contentType, payload, expiresAt, input.Now, input.Now)
	if err != nil {
		return browserauth.ResultExportDescriptor{}, safeDatabaseFailure("create browser result export", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return browserauth.ResultExportDescriptor{}, safeDatabaseFailure("read created browser result export id", err)
	}
	return browserauth.ResultExportDescriptor{ID: id, Format: input.Format, Status: "completed", Filename: filename, ContentType: contentType, URL: resultExportURL(id), ExpiresAt: &expiresAt, Authorized: true, Ready: true, ErrorMessage: nil}, nil
}

func (r *BrowserAuthRepository) GetResultExport(request *http.Request, actor browserauth.User, exportID int64) (browserauth.ResultExportDescriptor, error) {
	export, err := r.resultExport(request.Context(), actor.ActiveTenant(), exportID)
	if err != nil {
		return browserauth.ResultExportDescriptor{}, err
	}
	return export.descriptor(), nil
}

func (r *BrowserAuthRepository) DownloadResultExport(request *http.Request, actor browserauth.User, exportID int64, now time.Time) (browserauth.ResultExportDownload, error) {
	export, err := r.resultExport(request.Context(), actor.ActiveTenant(), exportID)
	if err != nil {
		return browserauth.ResultExportDownload{}, err
	}
	if export.Status != "completed" {
		return browserauth.ResultExportDownload{}, browserauth.ErrConflict
	}
	if export.ExpiresAt != nil && export.ExpiresAt.Before(now) {
		return browserauth.ResultExportDownload{}, browserauth.ErrGone
	}
	if export.Payload == nil {
		return browserauth.ResultExportDownload{}, browserauth.ErrConflict
	}
	return browserauth.ResultExportDownload{Filename: export.Filename, ContentType: export.ContentType, Payload: *export.Payload}, nil
}

func (r *BrowserAuthRepository) ListArtifactDescriptors(request *http.Request, actor browserauth.User, projectID int64, performedTestCycleID int64) ([]browserauth.ArtifactDescriptor, error) {
	rows, err := r.database.QueryContext(request.Context(), `SELECT id, idCostumer, idProject, performedTestCycleId, performedTestId, performedStepId, artifactType, name, contentType, sizeBytes, checksumSha256, storageKey, state, retentionUntil, metadata, created_at, updated_at
		FROM artifact_descriptors
		WHERE idCostumer = ? AND idProject = ? AND performedTestCycleId = ?
		ORDER BY artifactType ASC, name ASC`, actor.ActiveTenant(), projectID, performedTestCycleID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser artifact descriptors", err)
	}
	defer rows.Close()
	descriptors := []browserauth.ArtifactDescriptor{}
	for rows.Next() {
		descriptor, err := scanArtifactDescriptor(rows)
		if err != nil {
			return nil, err
		}
		descriptors = append(descriptors, descriptor)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser artifact descriptors", err)
	}
	return descriptors, nil
}

func (r *BrowserAuthRepository) GetArtifactDescriptor(request *http.Request, actor browserauth.User, projectID int64, performedTestCycleID int64, artifactDescriptorID int64) (browserauth.ArtifactDescriptor, error) {
	row := r.database.QueryRowContext(request.Context(), `SELECT id, idCostumer, idProject, performedTestCycleId, performedTestId, performedStepId, artifactType, name, contentType, sizeBytes, checksumSha256, storageKey, state, retentionUntil, metadata, created_at, updated_at
		FROM artifact_descriptors
		WHERE id = ? AND idCostumer = ? AND idProject = ? AND performedTestCycleId = ?
		LIMIT 1`, artifactDescriptorID, actor.ActiveTenant(), projectID, performedTestCycleID)
	descriptor, err := scanArtifactDescriptor(row)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ArtifactDescriptor{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.ArtifactDescriptor{}, err
	}
	return descriptor, nil
}

func (r *BrowserAuthRepository) RegisterArtifactDescriptor(request *http.Request, actor browserauth.User, input browserauth.ArtifactDescriptorCreate) (browserauth.ArtifactDescriptor, error) {
	if err := validateArtifactDescriptorCreate(input); err != nil {
		return browserauth.ArtifactDescriptor{}, err
	}
	ctx := request.Context()
	if _, err := r.performedCycle(ctx, actor.ActiveTenant(), input.PerformedTestCycleID); err != nil {
		return browserauth.ArtifactDescriptor{}, err
	}
	if input.State == "" {
		input.State = "available"
	}
	if input.RetentionUntil == nil {
		retentionUntil := input.Now.Add(30 * 24 * time.Hour)
		input.RetentionUntil = &retentionUntil
	}
	now := input.Now.UTC()
	metadata := nullableRawJSON(input.Metadata)
	result, err := r.database.ExecContext(ctx, `INSERT INTO artifact_descriptors (idCostumer, idProject, performedTestCycleId, performedTestId, performedStepId, artifactType, name, contentType, sizeBytes, checksumSha256, storageKey, state, retentionUntil, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		actor.ActiveTenant(), input.IDProject, input.PerformedTestCycleID, nullableInt64(input.PerformedTestID), nullableInt64(input.PerformedStepID), strings.TrimSpace(input.ArtifactType), strings.TrimSpace(input.Name), strings.TrimSpace(input.ContentType), input.SizeBytes, strings.ToLower(strings.TrimSpace(input.ChecksumSHA256)), strings.TrimSpace(input.StorageKey), strings.TrimSpace(input.State), input.RetentionUntil, metadata, now, now)
	if err != nil {
		return browserauth.ArtifactDescriptor{}, safeDatabaseFailure("register browser artifact descriptor", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return browserauth.ArtifactDescriptor{}, safeDatabaseFailure("read registered browser artifact descriptor id", err)
	}
	return r.GetArtifactDescriptor(request, actor, input.IDProject, input.PerformedTestCycleID, id)
}

func (r *BrowserAuthRepository) accountTarget(request *http.Request, actor browserauth.User, accountID int64) (browserauth.Account, error) {
	where, args := accountScope(actor)
	where += ` AND users.id = ?`
	args = append(args, accountID)
	page, err := r.ListAccounts(request, actor, browserauth.AccountQuery{Page: 1, PageSize: 1, Paged: false, Sort: "email", Direction: "asc", Search: ""})
	if err != nil {
		return browserauth.Account{}, err
	}
	_ = where
	_ = args
	for _, account := range page.Data {
		if account.ID == accountID {
			return account, nil
		}
	}
	return browserauth.Account{}, browserauth.ErrNotFound
}

func accountScope(actor browserauth.User) (string, []any) {
	if actor.Role == 1 {
		return `WHERE 1 = 1`, []any{}
	}
	return `WHERE users.role > 1 AND users.idCostumer = ?`, []any{actor.ActiveTenant()}
}

func countAccounts(ctx context.Context, database *sql.DB, where string, args []any) (int64, error) {
	var total int64
	err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM users JOIN costumers ON users.idCostumer = costumers.id JOIN roles ON users.role = roles.id `+where, args...).Scan(&total)
	if err != nil {
		return 0, safeDatabaseFailure("count browser accounts", err)
	}
	return total, nil
}

func accountSortColumn(sort string) string {
	switch sort {
	case "id":
		return "users.id"
	case "name":
		return "users.name"
	case "role":
		return "users.role"
	case "idCostumer":
		return "users.idCostumer"
	case "costumer":
		return "costumers.costumer"
	case "roleName":
		return "roles.name"
	default:
		return "users.email"
	}
}

func customerSortColumn(sort string) string {
	switch sort {
	case "id":
		return "id"
	case "costumer":
		return "costumer"
	case "description":
		return "description"
	case "licenseExpiration":
		return "licenseExpiration"
	case "updated_at":
		return "updated_at"
	default:
		return "created_at"
	}
}

func (r *BrowserAuthRepository) ensureProject(ctx context.Context, tenantID int64, projectID int64) error {
	return ensureProjectTx(ctx, r.database, tenantID, projectID)
}

func ensureProjectTx(ctx context.Context, execer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, tenantID int64, projectID int64) error {
	var exists int
	err := execer.QueryRowContext(ctx, `SELECT 1 FROM projects WHERE id = ? AND idCostumer = ? LIMIT 1`, projectID, tenantID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.ErrNotFound
	}
	if err != nil {
		return safeDatabaseFailure("check browser project ownership", err)
	}
	return nil
}

func resourceWhere(tenantID int64, projectID int64, query browserauth.ResourceQuery) (string, []any) {
	where := `WHERE idCostumer = ? AND idProject = ?`
	args := []any{tenantID, projectID}
	if query.Search != "" {
		where += ` AND (name LIKE ? OR description LIKE ?)`
		search := "%" + boundedSearch(query.Search) + "%"
		args = append(args, search, search)
	}
	if len(query.FilterIDs) > 0 {
		placeholders := make([]string, 0, len(query.FilterIDs))
		for _, id := range query.FilterIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		where += ` AND id IN (` + strings.Join(placeholders, ",") + `)`
	}
	return where, args
}

func countRows(ctx context.Context, database *sql.DB, table string, where string, args []any) (int64, error) {
	var total int64
	err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` `+where, args...).Scan(&total)
	if err != nil {
		return 0, safeDatabaseFailure("count browser resources", err)
	}
	return total, nil
}

func resourceSortColumn(sort string) string {
	switch sort {
	case "name":
		return "name"
	case "description":
		return "description"
	case "created_at":
		return "created_at"
	case "updated_at":
		return "updated_at"
	case "order":
		return "`order`"
	default:
		return "id"
	}
}

func resourceDirection(direction string) string {
	if direction == "desc" {
		return "DESC"
	}
	return "ASC"
}

func pageMeta(page int, pageSize int, total int64, sort string, direction string) browserauth.PageMeta {
	lastPage := int((total + int64(pageSize) - 1) / int64(pageSize))
	if lastPage == 0 {
		lastPage = 1
	}
	return browserauth.PageMeta{Page: page, PageSize: pageSize, Total: total, LastPage: lastPage, HasNextPage: page < lastPage, HasPreviousPage: page > 1, Sort: sort, Direction: direction, Stale: false, Partial: false}
}

func resultWhere(parentColumn string, tenantID int64, query browserauth.ResultQuery) (string, []any) {
	where := `WHERE idCostumer = ? AND ` + parentColumn + ` = ?`
	args := []any{tenantID, query.ParentID}
	if query.Status != nil {
		where += ` AND status = ?`
		args = append(args, *query.Status)
	}
	return where, args
}

func resultSortColumn(sort string, defaultSort string) string {
	switch sort {
	case "id":
		return "id"
	case "name":
		return "name"
	case "date":
		return "date"
	case "status":
		return "status"
	case "created_at":
		return "created_at"
	case "updated_at":
		return "updated_at"
	default:
		return defaultSort
	}
}

func resultMeta(page int, perPage int, total int64, sort string, direction string) browserauth.ResultMeta {
	lastPage := int((total + int64(perPage) - 1) / int64(perPage))
	if lastPage == 0 {
		lastPage = 1
	}
	return browserauth.ResultMeta{Pagination: browserauth.ResultPaginationMeta{Page: page, PerPage: perPage, Total: total, LastPage: lastPage, Sort: sort, Direction: direction}}
}

type performedCycleRecord struct {
	ID          int64
	TestCycleID int64
	Date        *time.Time
	Status      int64
}

type resultExportRecord struct {
	ID           int64
	Format       string
	Status       string
	Filename     string
	ContentType  string
	Payload      *string
	ExpiresAt    *time.Time
	ErrorMessage *string
}

func (record resultExportRecord) descriptor() browserauth.ResultExportDescriptor {
	return browserauth.ResultExportDescriptor{
		ID:           record.ID,
		Format:       record.Format,
		Status:       record.Status,
		Filename:     record.Filename,
		ContentType:  record.ContentType,
		URL:          resultExportURL(record.ID),
		ExpiresAt:    record.ExpiresAt,
		Authorized:   true,
		Ready:        record.Status == "completed",
		ErrorMessage: record.ErrorMessage,
	}
}

func (r *BrowserAuthRepository) performedCycle(ctx context.Context, tenantID int64, performedCycleID int64) (performedCycleRecord, error) {
	var record performedCycleRecord
	err := r.database.QueryRowContext(ctx, `SELECT id, testCycleId, date, status FROM performed_test_cycles WHERE id = ? AND idCostumer = ? LIMIT 1`, performedCycleID, tenantID).Scan(&record.ID, &record.TestCycleID, &record.Date, &record.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return performedCycleRecord{}, browserauth.ErrNotFound
	}
	if err != nil {
		return performedCycleRecord{}, safeDatabaseFailure("load browser result export run", err)
	}
	return record, nil
}

func (r *BrowserAuthRepository) resultExport(ctx context.Context, tenantID int64, exportID int64) (resultExportRecord, error) {
	var record resultExportRecord
	var payload sql.NullString
	var errorMessage sql.NullString
	err := r.database.QueryRowContext(ctx, `SELECT id, format, status, filename, contentType, payload, expiresAt, errorMessage FROM result_exports WHERE id = ? AND idCostumer = ? LIMIT 1`, exportID, tenantID).Scan(&record.ID, &record.Format, &record.Status, &record.Filename, &record.ContentType, &payload, &record.ExpiresAt, &errorMessage)
	if errors.Is(err, sql.ErrNoRows) {
		return resultExportRecord{}, browserauth.ErrNotFound
	}
	if err != nil {
		return resultExportRecord{}, safeDatabaseFailure("load browser result export", err)
	}
	if payload.Valid {
		record.Payload = &payload.String
	}
	if errorMessage.Valid && record.Status == "failed" {
		record.ErrorMessage = &errorMessage.String
	}
	return record, nil
}

func (r *BrowserAuthRepository) resultExportPayload(ctx context.Context, tenantID int64, run performedCycleRecord, format string) (string, error) {
	tests, err := r.resultExportTests(ctx, tenantID, run.ID)
	if err != nil {
		return "", err
	}
	document := map[string]any{
		"schemaVersion":        "result-export.v1",
		"performedTestCycleId": run.ID,
		"testCycleId":          run.TestCycleID,
		"status":               run.Status,
		"date":                 run.Date,
		"tests":                tests,
	}
	if format == "json" {
		encoded, err := json.MarshalIndent(document, "", "  ")
		if err != nil {
			return "", fmt.Errorf("encode browser result export payload: %w", err)
		}
		return string(encoded), nil
	}
	return resultExportMarkdown(document, tests), nil
}

func (r *BrowserAuthRepository) resultExportTests(ctx context.Context, tenantID int64, performedCycleID int64) ([]map[string]any, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT id, name, status FROM performed_tests WHERE testCycleDoneId = ? AND idCostumer = ? ORDER BY id ASC`, performedCycleID, tenantID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser result export tests", err)
	}
	defer rows.Close()
	tests := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		var status int64
		if err := rows.Scan(&id, &name, &status); err != nil {
			return nil, safeDatabaseFailure("scan browser result export test", err)
		}
		steps, err := r.resultExportSteps(ctx, tenantID, performedCycleID, id)
		if err != nil {
			return nil, err
		}
		tests = append(tests, map[string]any{"id": id, "name": name, "status": status, "steps": steps})
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser result export tests", err)
	}
	return tests, nil
}

func (r *BrowserAuthRepository) resultExportSteps(ctx context.Context, tenantID int64, performedCycleID int64, performedTestID int64) ([]map[string]any, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT id, name, status, type, created_at, updated_at FROM performed_steps WHERE testCycleDoneId = ? AND testDoneId = ? AND idCostumer = ? ORDER BY id ASC`, performedCycleID, performedTestID, tenantID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser result export steps", err)
	}
	defer rows.Close()
	steps := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		var status int64
		var stepType string
		var createdAt *time.Time
		var updatedAt *time.Time
		if err := rows.Scan(&id, &name, &status, &stepType, &createdAt, &updatedAt); err != nil {
			return nil, safeDatabaseFailure("scan browser result export step", err)
		}
		steps = append(steps, map[string]any{"id": id, "name": name, "status": status, "type": stepType, "created_at": createdAt, "updated_at": updatedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser result export steps", err)
	}
	return steps, nil
}

func resultExportMarkdown(document map[string]any, tests []map[string]any) string {
	lines := []string{
		"# Idelium execution report",
		"",
		fmt.Sprintf("- Schema: %s", document["schemaVersion"]),
		fmt.Sprintf("- Performed test cycle: %d", document["performedTestCycleId"]),
		fmt.Sprintf("- Source test cycle: %d", document["testCycleId"]),
		fmt.Sprintf("- Status: %d", document["status"]),
		fmt.Sprintf("- Date: %v", document["date"]),
		"",
		"## Tests",
		"",
	}
	for _, test := range tests {
		steps, _ := test["steps"].([]map[string]any)
		lines = append(lines,
			fmt.Sprintf("### %s", test["name"]),
			"",
			fmt.Sprintf("- Test ID: %d", test["id"]),
			fmt.Sprintf("- Status: %d", test["status"]),
			fmt.Sprintf("- Steps: %d", len(steps)),
			"",
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func resultExportFilename(performedCycleID int64, format string) string {
	extension := "md"
	if format == "json" {
		extension = "json"
	}
	return fmt.Sprintf("idelium-run-%d.%s", performedCycleID, extension)
}

func resultExportContentType(format string) string {
	if format == "json" {
		return "application/json"
	}
	return "text/markdown"
}

func resultExportURL(exportID int64) string {
	return fmt.Sprintf("/api/admin/result-exports/%d/download", exportID)
}

type artifactDescriptorScanner interface {
	Scan(dest ...any) error
}

func scanArtifactDescriptor(scanner artifactDescriptorScanner) (browserauth.ArtifactDescriptor, error) {
	var descriptor browserauth.ArtifactDescriptor
	var performedTestID sql.NullInt64
	var performedStepID sql.NullInt64
	var metadata sql.NullString
	if err := scanner.Scan(&descriptor.ID, &descriptor.IDCostumer, &descriptor.IDProject, &descriptor.PerformedTestCycleID, &performedTestID, &performedStepID, &descriptor.ArtifactType, &descriptor.Name, &descriptor.ContentType, &descriptor.SizeBytes, &descriptor.ChecksumSHA256, &descriptor.StorageKey, &descriptor.State, &descriptor.RetentionUntil, &metadata, &descriptor.CreatedAt, &descriptor.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return browserauth.ArtifactDescriptor{}, err
		}
		return browserauth.ArtifactDescriptor{}, safeDatabaseFailure("scan browser artifact descriptor", err)
	}
	if performedTestID.Valid {
		descriptor.PerformedTestID = &performedTestID.Int64
	}
	if performedStepID.Valid {
		descriptor.PerformedStepID = &performedStepID.Int64
	}
	if metadata.Valid {
		descriptor.Metadata = json.RawMessage(redactResultJSONString(metadata.String))
	}
	return descriptor, nil
}

func validateArtifactDescriptorCreate(input browserauth.ArtifactDescriptorCreate) error {
	missing := []string{}
	if input.IDProject <= 0 {
		missing = append(missing, "idProject")
	}
	if input.PerformedTestCycleID <= 0 {
		missing = append(missing, "performedTestCycleId")
	}
	if strings.TrimSpace(input.ArtifactType) == "" {
		missing = append(missing, "artifactType")
	}
	if strings.TrimSpace(input.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(input.ContentType) == "" {
		missing = append(missing, "contentType")
	}
	if input.SizeBytes == 0 {
		missing = append(missing, "sizeBytes")
	}
	if strings.TrimSpace(input.ChecksumSHA256) == "" {
		missing = append(missing, "checksumSha256")
	}
	if strings.TrimSpace(input.StorageKey) == "" {
		missing = append(missing, "storageKey")
	}
	if len(missing) > 0 {
		return fmt.Errorf("artifact descriptor fields are required: %s", strings.Join(missing, ", "))
	}
	if len(input.ChecksumSHA256) != 64 || !isHex(input.ChecksumSHA256) {
		return fmt.Errorf("artifact descriptor checksum must be a SHA-256 hex digest")
	}
	if len(input.Metadata) > 0 && !json.Valid(input.Metadata) {
		return fmt.Errorf("artifact descriptor metadata must be valid JSON")
	}
	return nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableRawJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func isHex(value string) bool {
	for _, character := range value {
		if (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f') || (character >= 'A' && character <= 'F') {
			continue
		}
		return false
	}
	return true
}

func redactResultJSONString(payload string) string {
	var value any
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return payload
	}
	encoded, err := json.Marshal(redactResultJSONValue(value))
	if err != nil {
		return payload
	}
	return string(encoded)
}

func redactResultJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveResultJSONKey(key) {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = redactResultJSONValue(child)
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, child := range typed {
			redacted[index] = redactResultJSONValue(child)
		}
		return redacted
	default:
		return typed
	}
}

func isSensitiveResultJSONKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	for _, marker := range []string{"authorization", "password", "secret", "token", "apikey", "api-key", "session", "csrf", "cookie"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func recordAssetVersion(ctx context.Context, execer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, actor browserauth.User, assetType string, assetID int64, projectID int64, reason string, snapshot map[string]any) error {
	var current sql.NullInt64
	if err := execer.QueryRowContext(ctx, `SELECT MAX(version) FROM asset_versions WHERE idCostumer = ? AND assetType = ? AND assetId = ?`, actor.ActiveTenant(), assetType, assetID).Scan(&current); err != nil {
		return safeDatabaseFailure("read browser asset version", err)
	}
	nextVersion := int64(1)
	if current.Valid {
		nextVersion = current.Int64 + 1
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode browser asset version snapshot: %w", err)
	}
	_, err = execer.ExecContext(ctx, `INSERT INTO asset_versions (idCostumer, idProject, assetType, assetId, version, actorUserId, reason, snapshot) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, actor.ActiveTenant(), projectID, assetType, assetID, nextVersion, actor.ID, reason, string(encoded))
	if err != nil {
		return safeDatabaseFailure("record browser asset version", err)
	}
	return nil
}

func boundedSearch(search string) string {
	search = strings.TrimSpace(search)
	if len(search) > 200 {
		return search[:200]
	}
	return search
}

func canManageAccount(actor browserauth.User, targetRole int64, targetTenant int64) bool {
	if actor.Role == 1 {
		return true
	}
	return actor.Role == 2 && targetRole > 1 && targetTenant == actor.ActiveTenant()
}

func existsByID(ctx context.Context, database *sql.DB, table string, id int64) (bool, error) {
	var exists int
	err := database.QueryRowContext(ctx, `SELECT 1 FROM `+table+` WHERE id = ? LIMIT 1`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, safeDatabaseFailure("check browser reference", err)
	}
	return true, nil
}

func requireAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return safeDatabaseFailure("count browser account mutation", err)
	}
	if affected == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func randomAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (r *BrowserAuthRepository) FindByEmail(ctx context.Context, email string) (browserauth.User, error) {
	var user browserauth.User
	err := r.database.QueryRowContext(ctx, `SELECT id, idCostumer, name, email, role, password, status FROM users WHERE email = ? LIMIT 1`, email).Scan(&user.ID, &user.TenantID, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.User{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.User{}, safeDatabaseFailure("find browser user", err)
	}
	return user, nil
}

func (r *BrowserAuthRepository) Create(ctx context.Context, session browserauth.Session) error {
	_, err := r.database.ExecContext(ctx, `INSERT INTO go_browser_sessions (idHash, userId, idCostumer, csrfTokenHash, expiresAt) VALUES (?, ?, ?, ?, ?)`, tokenHash(session.ID), session.UserID, session.TenantID, tokenHash(session.CSRFToken), session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create browser session: %w", safeDatabaseFailure("create browser session", err))
	}
	return nil
}

func (r *BrowserAuthRepository) Delete(ctx context.Context, sessionID string) error {
	result, err := r.database.ExecContext(ctx, `DELETE FROM go_browser_sessions WHERE idHash = ?`, tokenHash(sessionID))
	if err != nil {
		return fmt.Errorf("delete browser session: %w", safeDatabaseFailure("delete browser session", err))
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted browser session: %w", safeDatabaseFailure("count deleted browser session", err))
	}
	if affected == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func (r *BrowserAuthRepository) Get(ctx context.Context, sessionID string, now time.Time) (browserauth.User, error) {
	var user browserauth.User
	err := r.database.QueryRowContext(ctx, `SELECT u.id, u.idCostumer, COALESCE(sessions.activeTenantId, sessions.idCostumer), u.name, u.email, u.role, u.password, u.status, sessions.impersonationReason, sessions.impersonationExpiresAt
		FROM go_browser_sessions AS sessions
		JOIN users AS u ON u.id = sessions.userId AND u.idCostumer = sessions.idCostumer
		WHERE sessions.idHash = ? AND sessions.expiresAt > ? AND u.status = 'active'
		LIMIT 1`, tokenHash(sessionID), now).Scan(&user.ID, &user.TenantID, &user.ActiveTenantID, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.Status, &user.ImpersonationReason, &user.ImpersonationExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return browserauth.User{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.User{}, safeDatabaseFailure("load browser session", err)
	}
	return user, nil
}

func (r *BrowserAuthRepository) ListProjects(ctx context.Context, tenantID int64) ([]browserauth.Project, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at, idCostumer FROM projects WHERE idCostumer = ? ORDER BY created_at ASC`, tenantID)
	if err != nil {
		return nil, safeDatabaseFailure("list browser header projects", err)
	}
	defer rows.Close()

	projects := []browserauth.Project{}
	for rows.Next() {
		var project browserauth.Project
		if err := rows.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt, &project.IDCostumer); err != nil {
			return nil, safeDatabaseFailure("scan browser header projects", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser header projects", err)
	}
	return projects, nil
}

func (r *BrowserAuthRepository) ListCustomers(ctx context.Context) ([]browserauth.Customer, error) {
	rows, err := r.database.QueryContext(ctx, `SELECT id, costumer, description, licenseExpiration, created_at, updated_at FROM costumers ORDER BY created_at ASC`)
	if err != nil {
		return nil, safeDatabaseFailure("list browser header customers", err)
	}
	defer rows.Close()

	customers := []browserauth.Customer{}
	for rows.Next() {
		var customer browserauth.Customer
		if err := rows.Scan(&customer.ID, &customer.Costumer, &customer.Description, &customer.LicenseExpiration, &customer.CreatedAt, &customer.UpdatedAt); err != nil {
			return nil, safeDatabaseFailure("scan browser header customers", err)
		}
		customers = append(customers, customer)
	}
	if err := rows.Err(); err != nil {
		return nil, safeDatabaseFailure("read browser header customers", err)
	}
	return customers, nil
}

func (r *BrowserAuthRepository) CustomerExists(ctx context.Context, customerID int64) (bool, error) {
	var exists int
	err := r.database.QueryRowContext(ctx, `SELECT 1 FROM costumers WHERE id = ? LIMIT 1`, customerID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, safeDatabaseFailure("lookup browser target customer", err)
	}
	return true, nil
}

func (r *BrowserAuthRepository) SwitchTenant(ctx context.Context, tenantSwitch browserauth.TenantSwitch) error {
	result, err := r.database.ExecContext(ctx, `UPDATE go_browser_sessions
		SET activeTenantId = ?, impersonationReason = ?, impersonationExpiresAt = ?, updated_at = ?
		WHERE idHash = ? AND userId = ? AND idCostumer = ? AND expiresAt > ?`, tenantSwitch.ActiveTenant, tenantSwitch.Reason, tenantSwitch.ExpiresAt, tenantSwitch.Now, tokenHash(tenantSwitch.SessionID), tenantSwitch.UserID, tenantSwitch.ActorTenant, tenantSwitch.Now)
	if err != nil {
		return safeDatabaseFailure("switch browser tenant", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return safeDatabaseFailure("count browser tenant switch", err)
	}
	if affected == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func (r *BrowserAuthRepository) RecordTenantSwitch(ctx context.Context, event browserauth.AuditEvent) error {
	beforeValues, err := json.Marshal(event.BeforeValues)
	if err != nil {
		return fmt.Errorf("encode tenant switch audit before values: %w", err)
	}
	afterValues, err := json.Marshal(event.AfterValues)
	if err != nil {
		return fmt.Errorf("encode tenant switch audit after values: %w", err)
	}
	_, err = r.database.ExecContext(ctx, `INSERT INTO audit_events
		(actorUserId, actorTenantId, activeTenantId, action, targetType, targetId, beforeValues, afterValues, result, sourceIp, correlationId, metadata)
		VALUES (?, ?, ?, 'tenant.switch', 'costumer', ?, ?, ?, 'success', ?, ?, NULL)`, event.ActorUserID, event.ActorTenantID, event.ActiveTenantID, fmt.Sprint(event.TargetID), string(beforeValues), string(afterValues), event.SourceIP, event.CorrelationID)
	if err != nil {
		return safeDatabaseFailure("record tenant switch audit", err)
	}
	return nil
}

func tokenHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
