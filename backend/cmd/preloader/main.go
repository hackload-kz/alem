package main

import (
	"context"
	"log/slog"

	"hackload/internal/config"
	"hackload/internal/dependencies"
	"hackload/internal/service"
	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"
)

func main() {
	ctx := context.Background()

	conf, err := config.GetConfig(ctx)
	if err != nil {
		slog.Error("unable to get config", "error", err)
		return
	}

	deps, err := dependencies.NewDependencies(
		ctx,
		dependencies.WithDB(conf),
		dependencies.WithEventProvider(conf),
	)
	if err != nil {
		slog.Error("unable to get dependencies", "error", err)
		return
	}

	queries := sqlc.New(deps.DB)

	// Create EventProvider client with responses
	epClient, err := eventprovider.NewClientWithResponses(conf.EventProvider.Addr)
	if err != nil {
		slog.Error("unable to create event provider client", "error", err)
		return
	}

	if err := service.NewResetService(queries, deps.DB, epClient).Reset(ctx); err != nil {
		slog.Error("unable to reset", "error", err)
	}
}
