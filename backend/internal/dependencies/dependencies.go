package dependencies

import (
	"context"
	"database/sql"

	"hackload/internal/config"
	"hackload/pkg/eventprovider"
	"hackload/pkg/paymentgateway"

	_ "github.com/mattn/go-sqlite3"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
)

type Dependencies struct {
	DB             *sql.DB
	RiverDriver    *riversqlite.Driver
	RiverWorkers   *river.Workers
	RiverClient    *river.Client[*sql.Tx]
	EventProvider  eventprovider.ClientInterface
	PaymentGateway paymentgateway.ClientInterface
}

func (d *Dependencies) Close() {
	if d == nil {
		return
	}
	if d.DB != nil {
		d.DB.Close()
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
		db, err := sql.Open("sqlite3", conf.SQLite3Path)
		if err != nil {
			return err
		}
		d.DB = db
		return nil
	}
}

func WithRiverQueue(conf *config.Config) Option {
	return func(ctx context.Context, d *Dependencies) error {
		riverdriver := riversqlite.New(d.DB)

		migrator, err := rivermigrate.New(riverdriver, &rivermigrate.Config{})
		if err != nil {
			return err
		}

		if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
			return err
		}

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
