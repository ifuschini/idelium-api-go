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
