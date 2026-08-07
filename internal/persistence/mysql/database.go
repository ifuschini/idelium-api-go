package mysql

import (
	"context"
	"database/sql"
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
		return nil, fmt.Errorf("create MySQL connector: %w", err)
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
		return fmt.Errorf("MySQL readiness check failed: %w", err)
	}

	return nil
}
