package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"

	"github.com/go-sql-driver/mysql"

	"github.com/idelium/idelium-api-go/internal/config"
)

// Open creates a bounded MySQL connection pool without logging its DSN.
func Open(databaseConfig config.DatabaseConfig) (*sql.DB, error) {
	driverConfig := mysql.NewConfig()
	driverConfig.User = databaseConfig.User
	driverConfig.Passwd = databaseConfig.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = net.JoinHostPort(databaseConfig.Host, fmt.Sprintf("%d", databaseConfig.Port))
	driverConfig.DBName = databaseConfig.Name
	driverConfig.ParseTime = true
	driverConfig.Timeout = databaseConfig.ConnectTimeout
	driverConfig.ReadTimeout = databaseConfig.ReadTimeout
	driverConfig.WriteTimeout = databaseConfig.WriteTimeout
	driverConfig.TLSConfig = databaseConfig.TLSMode
	driverConfig.Collation = "utf8mb4_unicode_ci"

	connector, err := mysql.NewConnector(driverConfig)
	if err != nil {
		return nil, safeDatabaseFailure("create MySQL connector", err)
	}

	database := sql.OpenDB(connector)
	database.SetMaxOpenConns(databaseConfig.MaxOpenConnections)
	database.SetMaxIdleConns(databaseConfig.MaxIdleConnections)
	database.SetConnMaxLifetime(databaseConfig.ConnectionMaxLifetime)

	return database, nil
}

// Check verifies database availability within the caller's deadline.
func Check(ctx context.Context, database *sql.DB) error {
	if err := database.PingContext(ctx); err != nil {
		return safeDatabaseFailure("MySQL readiness check", err)
	}

	return nil
}

func safeDatabaseFailure(operation string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: deadline exceeded", operation)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: cancelled", operation)
	}
	var serverError *mysql.MySQLError
	if errors.As(err, &serverError) {
		return fmt.Errorf("%s: MySQL server error %d", operation, serverError.Number)
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		return fmt.Errorf("%s: network failure", operation)
	}
	return fmt.Errorf("%s: database unavailable", operation)
}
