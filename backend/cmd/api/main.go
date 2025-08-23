package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hackload/cmd/setup"
	"hackload/internal/config"
	"hackload/internal/dependencies"
	"hackload/internal/middleware"
	"hackload/internal/portriver"
	"hackload/internal/ports"
	"hackload/internal/sqlc"
	"hackload/pkg/telemetry"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx := context.Background()

	conf, err := config.GetConfig(ctx)
	if err != nil {
		slog.Error("unable to get config", "error", err)
		return
	}

	// Telemetry
	tel, shutdownTelemetry, err := telemetry.New(ctx, &telemetry.Config{
		Namespace:         conf.ServiceName,
		Service:           conf.ServiceName,
		Environment:       conf.Environment,
		OtelCollectorAddr: conf.OtelCollectorAddr,
	})
	if err != nil {
		slog.Error("unable to setup telemetry", "error", err)
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
		dependencies.WithEventProvider(conf),
		dependencies.WithPaymentGateway(conf),
		dependencies.WithAuthenticationService(),
		dependencies.WithResetService(conf),
	)
	if err != nil {
		slog.Error("unable to get dependencies", "error", err)
		return
	}

	queries := sqlc.New(deps.DB)

	// Workers use main DB for business logic, River uses separate DB for job queue
	river.AddWorker(
		deps.RiverWorkers,
		portriver.NewReleaseSeatsWorker(queries, deps.DB),
	)

	river.AddWorker(
		deps.RiverWorkers,
		portriver.NewSelectSeatsWorker(queries, deps.DB, deps.RiverClient, deps.EventProvider),
	)

	river.AddWorker(
		deps.RiverWorkers,
		portriver.NewConfirmOrderWorker(queries, deps.DB, deps.EventProvider),
	)

	river.AddWorker(
		deps.RiverWorkers,
		portriver.NewCancelBookingWorker(queries, deps.DB, deps.RiverClient, deps.EventProvider),
	)

	river.AddWorker(
		deps.RiverWorkers,
		portriver.NewRefundPaymentWorker(queries, deps.DB, deps.PaymentGateway, conf),
	)

	if err := deps.InitRiverClient(conf.River.MaxWorkers); err != nil {
		slog.Error("unable to init river client", "error", err)
		return
	}

	router := mux.NewRouter()
	router.Use(otelmux.Middleware(
		conf.ServiceName,
		otelmux.WithTracerProvider(tel.TraceProvider()),
	))

	ports.HandlerWithOptions(
		ports.NewHttpServer(
			queries,
			deps.DB,
			deps.RiverClient,
			deps.PaymentGateway,
			deps.ResetService,
			conf,
		), ports.GorillaServerOptions{
			BaseRouter:  router,
			Middlewares: []ports.MiddlewareFunc{},
		})

	// Now wrap specific routes with middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/payments") {
				middleware.AuthenticationMiddleware(deps.AuthenticationService)(next).ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	//////////////

	// Setup server

	server := &http.Server{
		Handler: handlers.CORS(
			handlers.AllowedHeaders([]string{"Authorization"}),
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		)(router),
		Addr: fmt.Sprintf(":%s", conf.API.Port),
	}

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, gCtx := errgroup.WithContext(mainCtx)

	// HTTP Server goroutine
	g.Go(func() error {
		slog.Info("starting HTTP server", "address", server.Addr)

		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed: %w", err)
		}

		slog.Info("HTTP server stopped accepting connections")
		return nil
	})

	// Signal handler goroutine
	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)

		select {
		case sig := <-c:
			slog.Info("received shutdown signal", "signal", sig)
			return fmt.Errorf("received signal: %s", sig)
		case <-gCtx.Done():
			return gCtx.Err()
		}
	})

	g.Go(func() error {
		<-gCtx.Done()
		slog.Info("shutting down telemetry")
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := shutdownTelemetry(shutdownCtx); err != nil {
			return fmt.Errorf("telemetry shutdown failed: %w", err)
		}

		return nil
	})

	// HTTP Server shutdown handler goroutine
	g.Go(func() error {
		<-gCtx.Done()

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 20*time.Second)
		defer shutdownRelease()

		slog.Info("shutting down HTTP server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}

		return nil
	})

	// RiverQueue
	g.Go(func() error {
		slog.Info("starting riverqueue")
		if err := deps.RiverClient.Start(gCtx); err != nil {
			return err
		}

		<-gCtx.Done()

		slog.Info("shutting down riverqueue")
		if err := deps.RiverClient.Stop(context.Background()); err != nil {
			return err
		}

		slog.Info("riverqueue stopped")

		return nil
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		slog.Error("application terminated", "error", err)
	}
}
