package mysql

import (
	"context"
	"database/sql"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

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
