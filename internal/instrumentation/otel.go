package instrumentation

import (
	"context"
	"errors"

	"github.com/nurhudajoantama/hmauto/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	zerolog "github.com/rs/zerolog/log"
)

type otl struct {
	resource *resource.Resource
	exporter *otlptrace.Exporter
	cfg      config.OTel
}

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, cfg config.OTel) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "hmauto"
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("0.0.1"),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		zerolog.Error().Err(err).Msg("failed to create OTEL resource")
	}

	o := &otl{resource: res, cfg: cfg}

	// Setup trace exporter: use OTLP if endpoint configured, else stdout.
	if cfg.Enabled && cfg.Endpoint != "" {
		traceExporter, err := otlptrace.New(
			ctx,
			otlptracegrpc.NewClient(
				otlptracegrpc.WithEndpoint(cfg.Endpoint),
				otlptracegrpc.WithInsecure(),
			),
		)
		if err != nil {
			return nil, err
		}
		o.exporter = traceExporter
	}

	prop := o.newPropagator()
	otel.SetTextMapPropagator(prop)

	tracerProvider, err := o.newTracerProvider()
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := o.newMeterProvider()
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	loggerProvider, err := o.newLoggerProvider()
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	return shutdown, err
}

func (o *otl) newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func (o *otl) newTracerProvider() (*trace.TracerProvider, error) {
	var batcher trace.TracerProviderOption

	if o.exporter != nil {
		batcher = trace.WithBatcher(o.exporter)
	} else {
		stdExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		batcher = trace.WithBatcher(stdExporter)
	}

	tracerProvider := trace.NewTracerProvider(
		batcher,
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(o.resource),
	)
	return tracerProvider, nil
}

func (o *otl) newMeterProvider() (*metric.MeterProvider, error) {
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(o.resource),
	)
	return meterProvider, nil
}

func (o *otl) newLoggerProvider() (*sdklog.LoggerProvider, error) {
	logExporter, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(o.resource),
	)
	return loggerProvider, nil
}
