package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type Config struct {
	Service           string
	Namespace         string
	Version           string
	Environment       string
	OtelCollectorAddr string
}

var (
	globalTelemetry = NewNull()

	// GenerateInstanceID how to generate instanceID
	// function open for changes
	instanceGenerator = genInstanceID
)

func Global() Telemetry {
	return globalTelemetry
}

func SetGlobal(t Telemetry) {
	globalTelemetry = t
}

type Telemetry struct {
	trace         trace.Tracer
	traceProvider trace.TracerProvider
}

func NewNull() Telemetry {
	return Telemetry{
		trace:         noop.NewTracerProvider().Tracer("noop"),
		traceProvider: noop.NewTracerProvider(),
	}
}

func New(ctx context.Context, cfg *Config) (tel Telemetry, shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	res := CreateRes(ctx, *cfg)

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTracerProvider(res, cfg)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	tel.traceProvider = tracerProvider
	tel.trace = otel.Tracer(fmt.Sprintf("%s_%s_tracer", cfg.Namespace, cfg.Service))

	SetGlobal(tel)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(res *resource.Resource, cfg *Config) (*sdktrace.TracerProvider, error) {
	traceExporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(cfg.OtelCollectorAddr),
			// otlptracegrpc.WithHeaders(headers),
			otlptracegrpc.WithInsecure(),
		),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			sdktrace.WithBatchTimeout(time.Second)),
	)
	return tracerProvider, nil
}

func genInstanceID(srv string) string {
	instSID := make([]byte, 4)
	_, _ = rand.Read(instSID)
	conv := hex.EncodeToString(instSID)

	instance := fmt.Sprintf("%s-%s", srv, conv)
	return instance
}

func CreateRes(ctx context.Context, l Config) *resource.Resource {
	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		// resource.WithProcess(),
		// resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends + tempo UI by this field perform service selection
			// key: service.name
			semconv.ServiceNameKey.String(l.Service),
			// we use tempo->loki reference in Grafana, but loki not support dots as it's in ServiceNameKey
			// in addition, we can't use service_name as it conflict with transformation to prometheus
			// key: service
			attribute.Key("service").String(l.Service),
			// key: service.namespace
			semconv.ServiceNamespaceKey.String(l.Namespace),
			// key: service.version
			semconv.ServiceVersionKey.String(l.Version),
			semconv.DeploymentEnvironmentKey.String(l.Environment),
			semconv.ServiceInstanceIDKey.String(instanceGenerator(l.Service)),
		),
	)

	return res
}

func (t Telemetry) T() trace.Tracer {
	return t.trace
}

func (t Telemetry) Tracer(name string, opts ...trace.TracerOption) Telemetry {
	t.trace = t.traceProvider.Tracer(name, opts...)
	return t
}

func (t Telemetry) TraceProvider() trace.TracerProvider {
	return t.traceProvider
}
