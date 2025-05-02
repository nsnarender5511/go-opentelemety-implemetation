package otel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/narender/common/config"
	logrustr "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func SetupLogrus(cfg *config.Config) *logrustr.Logger {
	logger := logrustr.New()

	level, err := logrustr.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Warnf("Invalid log level '%s', defaulting to 'info': %v", cfg.LogLevel, err)
		level = logrustr.InfoLevel
	}
	logger.SetLevel(level)

	switch strings.ToLower(cfg.LogFormat) {
	case "json":
		logger.SetFormatter(&logrustr.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "text":
		logger.SetFormatter(&logrustr.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		logger.Warnf("Invalid log format '%s', defaulting to 'text'", cfg.LogFormat)
		logger.SetFormatter(&logrustr.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	}

	logger.SetOutput(os.Stderr)

	logrustr.SetLevel(logger.GetLevel())
	logrustr.SetFormatter(logger.Formatter)
	logrustr.SetOutput(logger.Out)

	logger.Infof("Logrus initialized with level '%s' and format '%s'", logger.GetLevel(), cfg.LogFormat)
	return logger
}

func SetupOTelSDK(ctx context.Context, cfg *config.Config) (sdkLogger *logrustr.Logger, shutdown func(context.Context) error, err error) {
	logger := SetupLogrus(cfg)

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var shutdownErr error
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		err = errors.Join(err, shutdownErr)
		return shutdownErr
	}

	defer func() {
		if err != nil {
			logger.Debug("Attempting OTel cleanup after setup error...")
			shutdownErr := shutdown(context.Background())
			if shutdownErr != nil {
				logger.WithError(shutdownErr).Error("Error during OTel cleanup after setup failure")
			}
		} else {
			logger.Info("OTel SDK setup completed successfully.")
		}
	}()

	res, rErr := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithOS(),
	)
	if rErr != nil {
		err = fmt.Errorf("failed to create OTel resource: %w", rErr)
		logger.WithError(err).Error("OTel setup failed")
		return logger, shutdown, err
	}
	logger.Debug("OpenTelemetry resource created")

	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
	logger.Debug("OpenTelemetry propagation configured")

	exporterOpts := []grpc.DialOption{
	}
	var transportCreds credentials.TransportCredentials
	if cfg.OtelExporterInsecure {
		transportCreds = insecure.NewCredentials()
		logger.Warn("Using insecure connection for OTLP exporter")
	} else {
		logger.Warn("TLS configuration for OTLP exporter not implemented, using insecure connection as fallback.")
		transportCreds = insecure.NewCredentials()
	}
	exporterOpts = append(exporterOpts, grpc.WithTransportCredentials(transportCreds))

	traceClientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlptracegrpc.WithDialOption(exporterOpts...),
		otlptracegrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		traceClientOpts = append(traceClientOpts, otlptracegrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	traceExporter, texpErr := otlptracegrpc.New(ctx, traceClientOpts...)
	if texpErr != nil {
		err = fmt.Errorf("failed to create OTLP trace exporter: %w", texpErr)
		logger.WithError(err).Error("OTel setup failed")
		return logger, shutdown, err
	}
	logger.Info("OTLP trace exporter created")

	bspOpts := []trace.BatchSpanProcessorOption{
		trace.WithBatchTimeout(cfg.OtelBatchTimeout),
	}
	bsp := trace.NewBatchSpanProcessor(traceExporter, bspOpts...)

	sampler := trace.ParentBased(trace.TraceIDRatioBased(cfg.OtelSampleRatio))

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(sampler),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)
	logger.Info("Trace provider registered globally")

	metricClientOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlpmetricgrpc.WithDialOption(exporterOpts...),
		otlpmetricgrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		metricClientOpts = append(metricClientOpts, otlpmetricgrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	metricExporter, mexpErr := otlpmetricgrpc.New(ctx, metricClientOpts...)
	if mexpErr != nil {
		err = fmt.Errorf("failed to create OTLP metric exporter: %w", mexpErr)
		logger.WithError(err).Error("OTel setup failed")
		return logger, shutdown, err
	}
	logger.Info("OTLP metric exporter created")

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithTimeout(cfg.OtelExporterOtlpTimeout))),
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)
	logger.Info("Meter provider registered globally")

	return logger, shutdown, nil
}

func GetTracerProvider() oteltrace.TracerProvider {
	return otel.GetTracerProvider()
}

func GetMeterProvider() otelmetric.MeterProvider {
	return otel.GetMeterProvider()
}

func GetTracer(instrumentationName string) oteltrace.Tracer {
	return otel.Tracer(instrumentationName)
}

func GetMeter(instrumentationName string) otelmetric.Meter {
	return otel.Meter(instrumentationName)
}

func parseInsecure(insecureStr string) bool {
	val, err := strconv.ParseBool(strings.ToLower(insecureStr))
	if err != nil {
		logrustr.Warnf("Invalid boolean value for OTEL_EXPORTER_INSECURE: %s, defaulting to false", insecureStr)
		return false
	}
	return val
}
