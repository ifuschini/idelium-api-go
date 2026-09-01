package mysql

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/idelium/idelium-api-go/internal/browserauth"
)

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
