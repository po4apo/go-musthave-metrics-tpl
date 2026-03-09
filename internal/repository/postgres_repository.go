package repository

import (
	"context"
	"time"

	"go.uber.org/zap"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
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

func (r *PostgresRepository) SetStateFromSlice(*[]model.Metrics) error {
	return ErrNotEmplemented
}

func (r *PostgresRepository) String() (string, error) {
	return "", ErrNotEmplemented
}

func (r *PostgresRepository) IncreaseValue(*model.Metrics) error {
	return ErrNotEmplemented
}

func (r *PostgresRepository) ReplaceValue(*model.Metrics) error {
	return ErrNotEmplemented
}

func (r *PostgresRepository) GetMetric(string) (model.Metrics, error) {
	return model.Metrics{}, ErrNotEmplemented
}

func (r *PostgresRepository) GetAll() ([]model.Metrics, error) {
	return []model.Metrics{}, ErrNotEmplemented
}

func (r *PostgresRepository) Ping() error {
	ctx, canel := context.WithTimeout(context.Background(), 1*time.Second)
	defer canel()
	return r.db.PingContext(ctx)

}
