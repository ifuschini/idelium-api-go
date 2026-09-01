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

func (r *BrowserAuthRepository) DeleteProject(ctx context.Context, tenantID, projectID int64) error {
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return safeDatabaseFailure("start delete browser project", err)
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT 1 FROM projects WHERE id=? AND idCostumer=? FOR UPDATE", projectID, tenantID).Scan(&exists); err == sql.ErrNoRows {
		return browserauth.ErrNotFound
	} else if err != nil {
		return safeDatabaseFailure("lock browser project", err)
	}
	queries := []string{"DELETE FROM performed_steps WHERE idCostumer=? AND testCycleDoneId IN (SELECT id FROM performed_test_cycles WHERE idCostumer=? AND testCycleId IN (SELECT id FROM test_cycles WHERE idProject=? AND idCostumer=?))", "DELETE FROM performed_tests WHERE idCostumer=? AND testCycleDoneId IN (SELECT id FROM performed_test_cycles WHERE idCostumer=? AND testCycleId IN (SELECT id FROM test_cycles WHERE idProject=? AND idCostumer=?))", "DELETE FROM performed_test_cycles WHERE idCostumer=? AND testCycleId IN (SELECT id FROM test_cycles WHERE idProject=? AND idCostumer=?)", "DELETE FROM test_cycles WHERE idProject=? AND idCostumer=?", "DELETE FROM environments WHERE idProject=? AND idCostumer=?", "DELETE FROM plugins WHERE idProject=? AND idCostumer=?", "DELETE FROM steps WHERE idProject=? AND idCostumer=?", "DELETE FROM tests WHERE idProject=? AND idCostumer=?", "DELETE FROM projects WHERE id=? AND idCostumer=?"}
	args := [][]any{{tenantID, tenantID, projectID, tenantID}, {tenantID, tenantID, projectID, tenantID}, {tenantID, projectID, tenantID}, {projectID, tenantID}, {projectID, tenantID}, {projectID, tenantID}, {projectID, tenantID}, {projectID, tenantID}, {projectID, tenantID}, {projectID, tenantID}}
	for i, q := range queries {
		if _, err := tx.ExecContext(ctx, q, args[i]...); err != nil {
			return safeDatabaseFailure("delete browser project data", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return safeDatabaseFailure("commit delete browser project", err)
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
