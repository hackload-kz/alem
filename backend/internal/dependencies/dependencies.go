package dependencies

import (
	"context"
	"database/sql"
	"log/slog"

	"hackload/internal/config"
	"hackload/internal/service"
	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"
	"hackload/pkg/paymentgateway"

	_ "github.com/mattn/go-sqlite3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type Dependencies struct {
	DB                    *sql.DB
	RiverDB               *sql.DB
	RiverDriver           *riversqlite.Driver
	RiverWorkers          *river.Workers
	RiverClient           *river.Client[*sql.Tx]
	EventProvider         eventprovider.ClientInterface
	PaymentGateway        paymentgateway.ClientInterface
	AuthenticationService service.AuthenticationService
	ResetService          service.ResetService
}

func (d *Dependencies) Close() {
	if d == nil {
		return
	}
	if d.DB != nil {
		d.DB.Close()
	}
	if d.RiverDB != nil {
		d.RiverDB.Close()
	}
}

func NewDependencies(ctx context.Context, opts ...Option) (deps *Dependencies, err error) {
	defer func() {
		if err != nil {
			deps.Close()
		}
	}()

	deps = &Dependencies{}
	for _, opt := range opts {
		if err := opt(ctx, deps); err != nil {
			return nil, err
		}
	}

	return deps, nil
}

type Option func(context.Context, *Dependencies) error

func WithDB(conf *config.Config) Option {
	return func(ctx context.Context, d *Dependencies) error {
		// Apply SQLite performance optimizations based on https://turriate.com/articles/making-sqlite-faster-in-go
		// 1. Use optimized connection string with shared cache and WAL mode
		dsn := conf.SQLite3Path + "?cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-64000&_foreign_keys=1"

		db, err := otelsql.Open("sqlite3", dsn,
			otelsql.WithAttributes(semconv.DBSystemSqlite),
			otelsql.WithDBName(conf.SQLite3Path),
		)
		if err != nil {
			return err
		}

		// 2. Configure connection pool for better concurrency
		db.SetMaxOpenConns(10)   // Limit concurrent connections
		db.SetMaxIdleConns(5)    // Keep connections alive
		db.SetConnMaxLifetime(0) // No connection expiry (long-lived connections)
		db.SetConnMaxIdleTime(0) // No idle timeout

		// Test connection
		if err := db.Ping(); err != nil {
			db.Close()
			return err
		}

		// 3. Apply additional performance PRAGMA statements
		pragmas := []string{
			"PRAGMA temp_store = MEMORY",   // Store temp tables in memory
			"PRAGMA mmap_size = 268435456", // 256MB memory-mapped I/O
			"PRAGMA page_size = 32768",     // Larger page size for better performance
		}

		for _, pragma := range pragmas {
			if _, err := db.Exec(pragma); err != nil {
				slog.Warn("failed to set pragma", "pragma", pragma, "error", err)
				// Continue even if pragma fails - not critical
			}
		}

		d.DB = db
		return nil
	}
}

func WithRiverQueue(conf *config.Config) Option {
	return func(ctx context.Context, d *Dependencies) error {
		riverPath := conf.River.SQLite3Path

		// Apply SQLite performance optimizations for River database too
		riverDSN := riverPath + "?cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-32000&_foreign_keys=1"

		// Create separate database connection for River
		riverDB, err := sql.Open("sqlite3", riverDSN)
		if err != nil {
			return err
		}

		// Configure connection pool for River (smaller than main DB since it's job queue focused)
		riverDB.SetMaxOpenConns(5)
		riverDB.SetMaxIdleConns(2)
		riverDB.SetConnMaxLifetime(0)
		riverDB.SetConnMaxIdleTime(0)

		// Test the connection and ensure database is accessible
		if err := riverDB.Ping(); err != nil {
			riverDB.Close()
			return err
		}

		d.RiverDB = riverDB

		riverdriver := riversqlite.New(riverDB)

		migrator, err := rivermigrate.New(riverdriver, &rivermigrate.Config{})
		if err != nil {
			return err
		}

		// Run River migrations to create necessary tables
		slog.Info("running River migrations")
		migrationResult, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
		if err != nil {
			return err
		}

		slog.Info("River migrations completed", "versions_run", len(migrationResult.Versions))

		d.RiverDriver = riverdriver
		d.RiverWorkers = river.NewWorkers()

		return nil
	}
}

func (d *Dependencies) InitRiverClient(maxWorkers int) error {
	riverClient, err := river.NewClient(d.RiverDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: maxWorkers},
		},
		Workers: d.RiverWorkers,
	})
	if err != nil {
		return err
	}

	d.RiverClient = riverClient
	return nil
}

func WithAuthenticationService() Option {
	return func(ctx context.Context, d *Dependencies) error {
		queries := sqlc.New(d.DB)
		d.AuthenticationService = service.NewAuthenticationService(queries)
		return nil
	}
}

func WithResetService(conf *config.Config) Option {
	return func(ctx context.Context, d *Dependencies) error {
		queries := sqlc.New(d.DB)

		// Create EventProvider client with responses
		epClient, err := eventprovider.NewClientWithResponses(conf.EventProvider.Addr)
		if err != nil {
			return err
		}

		d.ResetService = service.NewResetService(
			queries,
			d.DB,
			epClient,
		)
		return nil
	}
}

func WithEventProvider(conf *config.Config) Option {
	return func(ctx context.Context, d *Dependencies) error {
		client, err := eventprovider.NewClient(conf.EventProvider.Addr)
		if err != nil {
			return err
		}
		d.EventProvider = client
		return nil
	}
}

func WithPaymentGateway(conf *config.Config) Option {
	return func(ctx context.Context, d *Dependencies) error {
		client, err := paymentgateway.NewClient(conf.PaymentProvider.Addr)
		if err != nil {
			return err
		}
		d.PaymentGateway = client
		return nil
	}
}
