package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/model"
	repoerrors "github.com/po4apo/go-musthave-metrics-tpl/internal/repository/errors"
	"github.com/po4apo/go-musthave-metrics-tpl/internal/retry"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// isPgRetriable определяет, стоит ли повторять запрос при данной PG-ошибке.
// Retriable: Class 08 (Connection Exception), 40001 (serialization_failure),
// 40P01 (deadlock_detected), 57P01 (admin_shutdown), 53300 (too_many_connections).
func isPgRetriable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch {
	case pgerrcode.IsConnectionException(pgErr.Code):
		return true
	case pgErr.Code == pgerrcode.SerializationFailure:
		return true
	case pgErr.Code == pgerrcode.DeadlockDetected:
		return true
	case pgErr.Code == pgerrcode.AdminShutdown:
		return true
	case pgErr.Code == pgerrcode.TooManyConnections:
		return true
	default:
		return false
	}
}

type PostgresRepository struct {
	db     *sql.DB
	logger *zap.Logger
	ctx    context.Context
	tables map[string]string
}

func (r *PostgresRepository) retryTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	return retry.Do(func() error {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if err := fn(tx); err != nil {
			return err
		}
		return tx.Commit()
	}, isPgRetriable)
}

// retryQuery выполняет read-only операцию (без транзакции) с retry при retriable PG-ошибках.
func (r *PostgresRepository) retryQuery(fn func() error) error {
	return retry.Do(fn, isPgRetriable)
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
		return errors.New("table metrics doesn't exist")
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

func resolveMigrationsPath() (string, error) {
	// Надежнее определять путь относительно файла, а не относительно текущей директории.
	_, file, _, ok := runtime.Caller(0)
	if ok {
		candidate := filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve migrations path: %w", err)
	}

	candidate := filepath.Join(wd, "migrations")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return candidate, nil
	}

	return "", fmt.Errorf("migrations directory not found")
}

func latestUpMigrationVersion(migrationsPath string) (uint, error) {
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return 0, fmt.Errorf("read migrations dir: %w", err)
	}

	var max uint
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		// Ожидаемый формат: 000001_metrics.up.sql
		base := strings.TrimSuffix(name, ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) == 0 {
			continue
		}

		v, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			continue
		}
		if v > uint64(max) {
			max = uint(v)
		}
	}

	return max, nil
}

func runMigrations(ctx context.Context, db *sql.DB, logger *zap.Logger) error {
	// ctx сейчас не используется напрямую библиотекой migrate, но оставляем
	// сигнатуру для возможных будущих изменений/логирования.
	_ = ctx

	migrationsPath, err := resolveMigrationsPath()
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("init postgres migrate driver: %w", err)
	}

	u := url.URL{Scheme: "file", Path: migrationsPath}
	sourceURL := u.String()

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}

	current, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		current = 0
		dirty = false
	} else if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database migration is dirty at version %d", current)
	}

	latest, err := latestUpMigrationVersion(migrationsPath)
	if err != nil {
		return fmt.Errorf("detect latest migration version: %w", err)
	}
	if latest == 0 {
		// На практике тут не должно быть, но пусть будет безопасно.
		logger.Info("No up migrations found; skip migrate", zap.String("migrationsPath", migrationsPath))
		return nil
	}

	if current >= latest {
		logger.Debug(
			"Database migrations are up to date",
			zap.Uint("version", current),
			zap.String("migrationsPath", migrationsPath),
		)
		return nil
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func NewPostgresStorage(ctx context.Context, logger *zap.Logger, databaseDsn string) (*PostgresRepository, error) {
	tables := make(map[string]string, 128)
	tables["metrics"] = "metrics"

	logger.Info(
		"Create db connection",
	)

	db, err := sql.Open("pgx", databaseDsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db connection: %w", err)
	}

	ctxP, cancel := context.WithCancel(ctx)
	defer cancel()
	err = db.PingContext(ctxP)
	if err != nil {
		return nil, err
	}
	logger.Info(
		"Db connection inizialized",
	)
	repo := &PostgresRepository{db, logger, ctx, tables}

	if err := runMigrations(ctx, db, logger); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run db migrations: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepository) SetStateFromSlice(state *[]model.Metrics) error {
	tableName := r.tables["metrics"]
	query := "INSERT INTO " + tableName + " (id, name, type, value, delta) " +
		"VALUES ($1, $2, $3, $4, $5)"

	for _, m := range *state {
		ctx, cancel := context.WithCancel(r.ctx)
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

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	return r.retryTx(ctx, func(tx *sql.Tx) error {
		var m model.Metrics
		row := tx.QueryRow("SELECT id, name, type, value, delta from "+r.tables["metrics"]+
			" WHERE id=$1;", metric.ID)
		if err := row.Scan(&m.ID, &m.Name, &m.MType, &m.Value, &m.Delta); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				query := "INSERT INTO " + r.tables["metrics"] + " (id, name, type, value, delta) " +
					"VALUES ($1, $2, $3, $4, $5)"
				_, err := tx.Exec(query, metric.ID, metric.Name, metric.MType, nil, *metric.Delta)
				return err
			}
			return err
		}
		newDelta := *m.Delta + *metric.Delta
		_, err := tx.Exec("UPDATE "+r.tables["metrics"]+" SET delta=$1 WHERE id=$2", newDelta, m.ID)
		return err
	})
}

func (r *PostgresRepository) ReplaceValue(metric *model.Metrics) error {
	if metric.MType != model.Gauge {
		return fmt.Errorf("failed replace %v by %v: %w", metric.Name, metric.Value, repoerrors.ErrUnsupportedType)
	}

	if metric.Value == nil {
		return fmt.Errorf("field \"Value\" is not define: %w", repoerrors.ErrFieldUndefine)
	}

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	return r.retryTx(ctx, func(tx *sql.Tx) error {
		query := "insert into " + r.tables["metrics"] + " (id, name, type, value)" +
			"values ($1, $2, $3, $4) on conflict(id) do update set value=$4;"
		_, err := tx.Exec(query, metric.ID, metric.Name, metric.MType, *metric.Value)
		return err
	})
}

func (r *PostgresRepository) BatchUpdate(metrics []model.Metrics) error {
	type gaugeRow struct {
		id, name, mtype string
		value           float64
	}
	type counterRow struct {
		id, name, mtype string
		delta           int64
	}

	// Предагрегация: gauge — последнее значение, counter — сумма дельт.
	// Порядок по ID гарантирует уникальность строк, чтобы INSERT ... ON CONFLICT
	// не получил дубликаты в одном VALUES (PostgreSQL запрещает это).
	gauges := make(map[string]gaugeRow)
	counters := make(map[string]counterRow)

	for i := range metrics {
		m := metrics[i]
		m.ID = model.GenerateID(m.MType, m.Name)

		switch m.MType {
		case model.Gauge:
			if m.Value == nil {
				return fmt.Errorf("field \"value\" is required for gauge %s", m.Name)
			}
			gauges[m.ID] = gaugeRow{id: m.ID, name: m.Name, mtype: m.MType, value: *m.Value}

		case model.Counter:
			if m.Delta == nil {
				return fmt.Errorf("field \"delta\" is required for counter %s", m.Name)
			}
			existing := counters[m.ID]
			existing.id = m.ID
			existing.name = m.Name
			existing.mtype = m.MType
			existing.delta += *m.Delta
			counters[m.ID] = existing

		default:
			return fmt.Errorf("unknown metric type: %s", m.MType)
		}
	}

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	return r.retryTx(ctx, func(tx *sql.Tx) error {
		tableName := r.tables["metrics"]

		if len(gauges) > 0 {
			query := "INSERT INTO " + tableName + " (id, name, type, value) VALUES "
			args := make([]interface{}, 0, len(gauges)*4)
			i := 0
			for _, g := range gauges {
				if i > 0 {
					query += ", "
				}
				base := i * 4
				query += fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4)
				args = append(args, g.id, g.name, g.mtype, g.value)
				i++
			}
			query += " ON CONFLICT(id) DO UPDATE SET value = EXCLUDED.value"

			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return fmt.Errorf("batch upsert gauges: %w", err)
			}
		}

		if len(counters) > 0 {
			query := "INSERT INTO " + tableName + " (id, name, type, delta) VALUES "
			args := make([]interface{}, 0, len(counters)*4)
			i := 0
			for _, c := range counters {
				if i > 0 {
					query += ", "
				}
				base := i * 4
				query += fmt.Sprintf("($%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4)
				args = append(args, c.id, c.name, c.mtype, c.delta)
				i++
			}
			query += " ON CONFLICT(id) DO UPDATE SET delta = COALESCE(" + tableName + ".delta, 0) + EXCLUDED.delta"

			if _, err := tx.ExecContext(ctx, query, args...); err != nil {
				return fmt.Errorf("batch upsert counters: %w", err)
			}
		}

		return nil
	})
}

func (r *PostgresRepository) GetMetric(id string) (model.Metrics, error) {
	var m model.Metrics
	err := r.retryQuery(func() error {
		ctx, cancel := context.WithCancel(r.ctx)
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
	})

	return m, err
}

func (r *PostgresRepository) GetAll() ([]model.Metrics, error) {
	var result []model.Metrics
	err := r.retryQuery(func() error {
		ctx, cancel := context.WithCancel(r.ctx)
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
	})

	if err != nil {
		return []model.Metrics{}, err
	}
	return result, nil
}

func (r *PostgresRepository) Ping() error {
	ctx, canel := context.WithCancel(r.ctx)
	defer canel()
	return r.db.PingContext(ctx)
}
