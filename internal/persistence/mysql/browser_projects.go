package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

func (r *BrowserAuthRepository) CreateProject(ctx context.Context, tenantID int64, name, description string) error {
	now := time.Now().UTC()
	_, err := r.database.ExecContext(ctx, "INSERT INTO projects (name,description,idCostumer,created_at,updated_at) VALUES (?,?,?,?,?)", name, description, tenantID, now, now)
	if err != nil {
		return safeDatabaseFailure("create browser project", err)
	}
	return nil
}
func (r *BrowserAuthRepository) UpdateProject(ctx context.Context, tenantID, projectID int64, name, description string) error {
	res, err := r.database.ExecContext(ctx, "UPDATE projects SET name=?,description=?,updated_at=? WHERE id=? AND idCostumer=? AND archivedAt IS NULL", name, description, time.Now().UTC(), projectID, tenantID)
	if err != nil {
		return safeDatabaseFailure("update browser project", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func (r *BrowserAuthRepository) GetProject(ctx context.Context, tenantID, projectID int64) (browserauth.Project, error) {
	var p browserauth.Project
	var description sql.NullString
	err := r.database.QueryRowContext(ctx, "SELECT id, name, description, idCostumer, created_at, updated_at FROM projects WHERE id=? AND idCostumer=? AND archivedAt IS NULL LIMIT 1", projectID, tenantID).Scan(&p.ID, &p.Name, &description, &p.IDCostumer, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return browserauth.Project{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.Project{}, safeDatabaseFailure("show browser project", err)
	}
	if description.Valid {
		p.Description = &description.String
	}
	return p, nil
}
