package repository

import (
	"context"
	"time"

	"go.uber.org/zap"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewPostgresStorage(logger *zap.Logger, databaseDsn string) (*PostgresRepository, error) {
	logger = logger.With(zap.String("component", "PostgresStorage"))
	logger.Info(
		"Create db connection",
		zap.String("databaseDsn", databaseDsn),
	)

	db, err := sql.Open("pgx", databaseDsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	logger.Info(
		"Db connection inizialized",
		zap.String("databaseDsn", databaseDsn),
	)

	return &PostgresRepository{db, logger}, nil
}

func (repo *PostgresRepository) Ping() error {
	ctx, canel := context.WithTimeout(context.Background(), 1*time.Second)
	defer canel()
	return repo.db.PingContext(ctx)

}
