package mysql

import (
	"context"
	"database/sql"
	"errors"
)

const integrationWorkerLockName = "idelium.integration-deliveries.worker.v1"

type IntegrationWorkerLease struct {
	connection *sql.Conn
}

func AcquireIntegrationWorkerLease(ctx context.Context, database *sql.DB) (*IntegrationWorkerLease, error) {
	if database == nil {
		return nil, errors.New("integration worker database is required")
	}
	connection, err := database.Conn(ctx)
	if err != nil {
		return nil, errors.New("integration worker lease is unavailable")
	}
	var acquired sql.NullInt64
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", integrationWorkerLockName).Scan(&acquired); err != nil || !acquired.Valid || acquired.Int64 != 1 {
		_ = connection.Close()
		return nil, errors.New("another integration delivery worker is active")
	}
	return &IntegrationWorkerLease{connection: connection}, nil
}

func (lease *IntegrationWorkerLease) Release(ctx context.Context) error {
	if lease == nil || lease.connection == nil {
		return nil
	}
	var released sql.NullInt64
	err := lease.connection.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", integrationWorkerLockName).Scan(&released)
	closeErr := lease.connection.Close()
	lease.connection = nil
	if err != nil || !released.Valid || released.Int64 != 1 {
		return errors.New("integration worker lease could not be released cleanly")
	}
	if closeErr != nil {
		return errors.New("integration worker lease connection could not be closed cleanly")
	}
	return nil
}
