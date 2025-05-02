package telemetry

import (
	"context"
	"errors"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/exporter"
	"github.com/narender/common/telemetry/manager"
	"github.com/narender/common/telemetry/metric"
	"github.com/narender/common/telemetry/propagator"
	"github.com/narender/common/telemetry/resource"
	"github.com/narender/common/telemetry/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {

	
	tempLogger := zap.NewExample() 
	defer tempLogger.Sync()

	var shutdownFuncs []func(context.Context) error
	var tp *sdktrace.TracerProvider = nil
	var mp *sdkmetric.MeterProvider = nil

	shutdown = func(ctx context.Context) error {
		var shutdownErr error

		if mp != nil {
			tempLogger.Debug("Shutting down MeterProvider...")
			shutdownErr = errors.Join(shutdownErr, mp.Shutdown(ctx))
		}
		if tp != nil {
			tempLogger.Debug("Shutting down TracerProvider...")
			shutdownErr = errors.Join(shutdownErr, tp.Shutdown(ctx))
		}

		tempLogger.Debug("Executing additional shutdown functions", zap.Int("count", len(shutdownFuncs)))
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		tempLogger.Info("OpenTelemetry resources shut down sequence completed.")
		return shutdownErr
	}

	defer func() {
		if err != nil {
			tempLogger.Error("OpenTelemetry SDK initialization failed", zap.Error(err))
			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
				tempLogger.Error("Error during OTel cleanup after setup failure", zap.Error(shutdownErr))
			}
			manager.InitializeGlobalManager(nil, nil, nil, cfg.ServiceName, cfg.ServiceVersion)
		}
	}()

	res, err := resource.NewResource(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create resource: %w", err)
	}

	propagator.SetupPropagators()

	traceExporter, err := exporter.NewTraceExporter(ctx, cfg, tempLogger)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	sampler := trace.NewSampler(cfg)
	var bspShutdown func(context.Context) error
	tp, bspShutdown = trace.NewTraceProvider(res, traceExporter, sampler)
	if bspShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, bspShutdown)
	}
	otel.SetTracerProvider(tp)
	tempLogger.Debug("Standard TracerProvider initialized and set globally.")

	metricExporter, err := exporter.NewMetricExporter(ctx, cfg, tempLogger)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create metric exporter: %w", err)
	}
	mp = metric.NewMeterProvider(cfg, res, metricExporter)
	otel.SetMeterProvider(mp)
	tempLogger.Debug("Standard MeterProvider initialized and set globally.")

	manager.InitializeGlobalManager(tp, mp, nil, cfg.ServiceName, cfg.ServiceVersion)
	tempLogger.Info("Global TelemetryManager initialized successfully (without logger).")

	return shutdown, nil
}
