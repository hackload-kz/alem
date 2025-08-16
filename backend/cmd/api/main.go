package main

import (
	"context"
	"log/slog"

	"hackload/cmd/setup"
	"hackload/internal/config"
	"hackload/internal/dependencies"
)

func main() {
	ctx := context.Background()

	conf, err := config.GetConfig(ctx)
	if err != nil {
		slog.Error("unable to get config", "error", err)
		return
	}

	args := config.ParseArgs()

	if err := setup.Setup(ctx, conf, args); err != nil {
		slog.Error("unable to setup", "error", err)
		return
	}

	deps, err := dependencies.NewDependencies(
		ctx,
		dependencies.WithDB(conf),
		dependencies.WithRiverQueue(conf),
	)
	if err != nil {
		slog.Error("unable to get dependencies", "error", err)
		return
	}

	if err := deps.InitRiverClient(conf.RiverMaxWorkers); err != nil {
		slog.Error("unable to init river client", "error", err)
		return
	}

	_ = deps
}
