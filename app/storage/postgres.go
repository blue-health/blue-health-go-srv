package storage

import (
	"context"
	"database/sql"
	"fmt"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/blue-health/blue-health-go-srv/app/util"
	"github.com/jmoiron/sqlx"
)

const (
	driver         = "postgres"
	cloudSQLDriver = "cloudsql-pq"
)

const pageSize = 20

func init() {
	sqlx.BindDriver(cloudSQLDriver, sqlx.DOLLAR)
}

func NewPostgres(ctx context.Context, dsn string, maxConns int) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(maxConns)

	return db, nil
}

func NewCloudPostgres(ctx context.Context, dsn, cloudName string, maxConns int) (*sqlx.DB, func() error, error) {
	d, cleanup, err := util.NewCloudSQLDriver(ctx, cloudName, cloudsqlconn.WithDefaultDialOptions(
		cloudsqlconn.WithPrivateIP(),
	))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cloud driver: %w", err)
	}

	sql.Register(cloudSQLDriver, d)

	db, err := sqlx.ConnectContext(ctx, cloudSQLDriver, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(maxConns)

	return db, cleanup, nil
}
