package dependencies

import (
	"context"
	"database/sql"

	"hackload/pkg/eventprovider"
	"hackload/pkg/paymentgateway"

	_ "github.com/mattn/go-sqlite3"
)

type Dependencies struct {
	DB             *sql.DB
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

func WithDB(sqlitePath string) Option {
	return func(ctx context.Context, d *Dependencies) error {
		db, err := sql.Open("sqlite3", sqlitePath)
		if err != nil {
			return err
		}
		d.DB = db
		return nil
	}
}
