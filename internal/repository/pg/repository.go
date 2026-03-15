package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"database/sql"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	repoerrors "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/errors"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/retry"
)

const (
	InsertRowTimeout     = 5 * time.Second
	SelectRowTimeout     = 5 * time.Second
	IncreaseValueTimeout = 5 * time.Second
	ReplaceValueTimeout  = 5 * time.Second
)

// isPgConnError возвращает true, если ошибка относится к Class 08 — Connection Exception.
func isPgConnError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	return false
}

type PostgresRepository struct {
	db     *sql.DB
	logger *zap.Logger
	ctx    context.Context
	tables map[string]string
}

func (r PostgresRepository) changeTable(key, value string) error {
	if _, ok := r.tables[key]; ok {
		r.tables[key] = value
		return nil
	}
	return fmt.Errorf("table %s doesn't exist", key)
}

func (r PostgresRepository) createMetricsTable(ctx context.Context) error {
	tableName, ok := r.tables["metrics"]
	if !ok {
		return fmt.Errorf("table metrics doesn't exist")
	}

	query := "CREATE TABLE IF NOT EXISTS " + tableName + " " +
		"(id VARCHAR(255) PRIMARY KEY," +
		"name VARCHAR(255) NOT NULL," +
		"type VARCHAR(255) NOT NULL," +
		"value DOUBLE PRECISION," +
		"delta BIGINT);"

	_, err := r.db.ExecContext(ctx, query)
	return err
}

func NewPostgresStorage(ctx context.Context, logger *zap.Logger, databaseDsn string) (*PostgresRepository, error) {
	tables := make(map[string]string, 128)
	tables["metrics"] = "metrics"

	logger = logger.With(zap.String("component", "PostgresStorage"))
	logger.Info(
		"Create db connection",
		zap.String("databaseDsn", databaseDsn),
	)

	db, err := sql.Open("pgx", databaseDsn)
	if err != nil {
		return nil, err
	}

	ctxP, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = db.PingContext(ctxP)
	if err != nil {
		return nil, err
	}
	logger.Info(
		"Db connection inizialized",
		zap.String("databaseDsn", databaseDsn),
	)
	repo := &PostgresRepository{db, logger, ctx, tables}

	if err := repo.createMetricsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create metrics table: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepository) SetStateFromSlice(state *[]model.Metrics) error {
	tableName := r.tables["metrics"]
	query := "INSERT INTO " + tableName + " (id, name, type, value, delta) " +
		"VALUES ($1, $2, $3, $4, $5)"

	for _, m := range *state {
		ctx, cancel := context.WithTimeout(r.ctx, InsertRowTimeout)
		defer cancel()
		var value sql.NullFloat64
		if m.Value != nil {
			value = sql.NullFloat64{Float64: *m.Value, Valid: true}
		}
		var delta sql.NullInt64
		if m.Delta != nil {
			delta = sql.NullInt64{Int64: *m.Delta, Valid: true}
		}

		_, err := r.db.ExecContext(ctx, query,
			m.ID, m.Name, m.MType, value, delta)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) String() (string, error) {
	metrics, err := r.GetAll()
	if err != nil {
		return "", nil
	}
	b, err := json.MarshalIndent(metrics, "", "  ")

	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (r *PostgresRepository) IncreaseValue(metric *model.Metrics) error {
	if metric.MType != model.Counter {
		return fmt.Errorf("failed increase %v by %v: %w", metric.Name, metric.Value, repoerrors.ErrUnsupportedType)
	}
	if metric.Delta == nil {
		return fmt.Errorf("field \"Delta\" is not define: %w", repoerrors.ErrFieldUndefine)
	}

	return retry.Do(func() error {
		ctx, cancel := context.WithTimeout(r.ctx, IncreaseValueTimeout)
		defer cancel()
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		var m model.Metrics
		row := tx.QueryRow("SELECT id, name, type, value, delta from "+r.tables["metrics"]+
			" WHERE id=$1;", metric.ID)
		if err := row.Scan(&m.ID, &m.Name, &m.MType, &m.Value, &m.Delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				query := "INSERT INTO " + r.tables["metrics"] + " (id, name, type, value, delta) " +
					"VALUES ($1, $2, $3, $4, $5)"
				if _, err := tx.Exec(query, metric.ID, metric.Name, metric.MType, nil, *metric.Delta); err != nil {
					return err
				}
				return tx.Commit()
			}
			return err
		}
		newDelta := *m.Delta + *metric.Delta
		if _, err = tx.Exec("UPDATE "+r.tables["metrics"]+" SET delta=$1 WHERE id=$2", newDelta, m.ID); err != nil {
			return err
		}
		return tx.Commit()
	}, isPgConnError)
}

func (r *PostgresRepository) ReplaceValue(metric *model.Metrics) error {
	if metric.MType != model.Gauge {
		return fmt.Errorf("failed replace %v by %v: %w", metric.Name, metric.Value, repoerrors.ErrUnsupportedType)
	}

	if metric.Value == nil {
		return fmt.Errorf("field \"Value\" is not define: %w", repoerrors.ErrFieldUndefine)
	}

	return retry.Do(func() error {
		ctx, cancel := context.WithTimeout(r.ctx, ReplaceValueTimeout)
		defer cancel()
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		query := "insert into " + r.tables["metrics"] + " (id, name, type, value)" +
			"values ($1, $2, $3, $4) on conflict(id) do update set value=$4;"

		if _, err = tx.Exec(query, metric.ID, metric.Name, metric.MType, *metric.Value); err != nil {
			return err
		}
		return tx.Commit()
	}, isPgConnError)
}

const BatchUpdateTimeout = 10 * time.Second

func (r *PostgresRepository) BatchUpdate(metrics []model.Metrics) error {
	return retry.Do(func() error {
		ctx, cancel := context.WithTimeout(r.ctx, BatchUpdateTimeout)
		defer cancel()

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback()

		tableName := r.tables["metrics"]

		gaugeStmt, err := tx.PrepareContext(ctx,
			"INSERT INTO "+tableName+" (id, name, type, value) "+
				"VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO UPDATE SET value = $4")
		if err != nil {
			return fmt.Errorf("prepare gauge stmt: %w", err)
		}
		defer gaugeStmt.Close()

		counterSelectStmt, err := tx.PrepareContext(ctx,
			"SELECT delta FROM "+tableName+" WHERE id = $1")
		if err != nil {
			return fmt.Errorf("prepare counter select stmt: %w", err)
		}
		defer counterSelectStmt.Close()

		counterInsertStmt, err := tx.PrepareContext(ctx,
			"INSERT INTO "+tableName+" (id, name, type, delta) "+
				"VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO UPDATE SET delta = $4")
		if err != nil {
			return fmt.Errorf("prepare counter insert stmt: %w", err)
		}
		defer counterInsertStmt.Close()

		for i := range metrics {
			m := metrics[i]
			m.ID = model.GenerateID(m.MType, m.Name)

			switch m.MType {
			case model.Gauge:
				if m.Value == nil {
					return fmt.Errorf("field \"value\" is required for gauge %s", m.Name)
				}
				if _, err := gaugeStmt.ExecContext(ctx, m.ID, m.Name, m.MType, *m.Value); err != nil {
					return fmt.Errorf("upsert gauge %s: %w", m.Name, err)
				}

			case model.Counter:
				if m.Delta == nil {
					return fmt.Errorf("field \"delta\" is required for counter %s", m.Name)
				}
				var existingDelta sql.NullInt64
				err := counterSelectStmt.QueryRowContext(ctx, m.ID).Scan(&existingDelta)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("select counter %s: %w", m.Name, err)
				}
				newDelta := *m.Delta
				if existingDelta.Valid {
					newDelta += existingDelta.Int64
				}
				if _, err := counterInsertStmt.ExecContext(ctx, m.ID, m.Name, m.MType, newDelta); err != nil {
					return fmt.Errorf("upsert counter %s: %w", m.Name, err)
				}

			default:
				return fmt.Errorf("unknown metric type: %s", m.MType)
			}
		}

		return tx.Commit()
	}, isPgConnError)
}

func (r *PostgresRepository) GetMetric(id string) (model.Metrics, error) {
	var m model.Metrics
	err := retry.Do(func() error {
		ctx, cancel := context.WithTimeout(r.ctx, SelectRowTimeout)
		defer cancel()

		query := "SELECT id, name, type, value, delta from " + r.tables["metrics"] +
			" WHERE id = $1;"

		row := r.db.QueryRowContext(ctx, query, id)
		if err := row.Scan(&m.ID, &m.Name, &m.MType, &m.Value, &m.Delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return repoerrors.ErrNotFound
			}
			return err
		}
		return nil
	}, isPgConnError)

	return m, err
}

func (r *PostgresRepository) GetAll() ([]model.Metrics, error) {
	var result []model.Metrics
	err := retry.Do(func() error {
		ctx, cancel := context.WithTimeout(r.ctx, SelectRowTimeout)
		defer cancel()

		rows, err := r.db.QueryContext(ctx, "SELECT * FROM "+r.tables["metrics"])
		if err != nil {
			return err
		}
		defer rows.Close()

		metrics := make([]model.Metrics, 0, 128)
		for rows.Next() {
			var m model.Metrics
			if err := rows.Scan(&m.ID, &m.Name, &m.MType, &m.Value, &m.Delta); err != nil {
				return err
			}
			metrics = append(metrics, m)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		result = metrics
		return nil
	}, isPgConnError)

	if err != nil {
		return []model.Metrics{}, err
	}
	return result, nil
}

func (r *PostgresRepository) Ping() error {
	ctx, canel := context.WithTimeout(context.Background(), 1*time.Second)
	defer canel()
	return r.db.PingContext(ctx)
}
