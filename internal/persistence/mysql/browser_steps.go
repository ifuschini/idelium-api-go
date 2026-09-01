package mysql

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

func stepConfig(v any) string {
	b, _ := json.Marshal(v)
	if v == nil {
		return "{}"
	}
	return string(b)
}
func (r *BrowserAuthRepository) CreateStep(req *http.Request, actor browserauth.User, in browserauth.StepInput) error {
	ctx := req.Context()
	if err := r.ensureProject(ctx, actor.ActiveTenant(), in.IDProject); err != nil {
		return err
	}
	_, err := r.database.ExecContext(ctx, "INSERT INTO steps (name,description,config,idProject,idCostumer,`order`,created_at,updated_at) VALUES (?,?,?,?,?,9999999,?,?)", strings.TrimSpace(in.Name), in.Description, stepConfig(in.Config), in.IDProject, actor.ActiveTenant(), time.Now().UTC(), time.Now().UTC())
	if err != nil {
		return safeDatabaseFailure("create browser step", err)
	}
	return nil
}
func (r *BrowserAuthRepository) UpdateStep(req *http.Request, actor browserauth.User, in browserauth.StepInput) error {
	res, err := r.database.ExecContext(req.Context(), "UPDATE steps SET name=?,description=?,config=?,updated_at=? WHERE id=? AND idProject=? AND idCostumer=?", in.Name, in.Description, stepConfig(in.Config), time.Now().UTC(), in.ID, in.IDProject, actor.ActiveTenant())
	if err != nil {
		return safeDatabaseFailure("update browser step", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}
func (r *BrowserAuthRepository) DeleteStep(req *http.Request, actor browserauth.User, projectID, stepID int64) error {
	res, err := r.database.ExecContext(req.Context(), "DELETE FROM steps WHERE id=? AND idProject=? AND idCostumer=?", stepID, projectID, actor.ActiveTenant())
	if err != nil {
		return safeDatabaseFailure("delete browser step", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return browserauth.ErrNotFound
	}
	return nil
}

func (r *BrowserAuthRepository) GetStep(request *http.Request, actor browserauth.User, projectID, stepID int64) (browserauth.StepDetail, error) {
	var step browserauth.StepDetail
	var config sql.NullString
	err := r.database.QueryRowContext(request.Context(), "SELECT id, name, description, config, idProject, `order` FROM steps WHERE id = ? AND idProject = ? AND idCostumer = ? LIMIT 1", stepID, projectID, actor.ActiveTenant()).Scan(&step.ID, &step.Name, &step.Description, &config, &step.IDProject, &step.Order)
	if err == sql.ErrNoRows {
		return browserauth.StepDetail{}, browserauth.ErrNotFound
	}
	if err != nil {
		return browserauth.StepDetail{}, safeDatabaseFailure("show browser step", err)
	}
	step.Config = map[string]any{}
	if config.Valid && config.String != "" {
		if err := json.Unmarshal([]byte(config.String), &step.Config); err != nil {
			return browserauth.StepDetail{}, safeDatabaseFailure("decode browser step config", err)
		}
	}
	return step, nil
}
