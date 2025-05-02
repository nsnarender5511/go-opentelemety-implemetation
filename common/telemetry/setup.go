package telemetry

import (
	"context"
	"errors"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/logging"
	"github.com/narender/common/telemetry/exporter"
	logotel "github.com/narender/common/telemetry/log"
	"github.com/narender/common/telemetry/manager"
	"github.com/narender/common/telemetry/metric"
	"github.com/narender/common/telemetry/propagator"
	"github.com/narender/common/telemetry/resource"
	"github.com/narender/common/telemetry/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)


func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {

	logger, err := logging.InitZapLogger(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize base logger for telemetry setup: %w", err)
	}

	var shutdownFuncs []func(context.Context) error
	var tp *sdktrace.TracerProvider = nil
	var mp *sdkmetric.MeterProvider = nil
	var lp *sdklog.LoggerProvider = nil

	shutdown = func(ctx context.Context) error {

		var shutdownErr error
		if lp != nil {
			logger.Debug("Shutting down LoggerProvider...")
			shutdownErr = errors.Join(shutdownErr, lp.Shutdown(ctx))
		}
		if mp != nil {
			logger.Debug("Shutting down MeterProvider...")
			shutdownErr = errors.Join(shutdownErr, mp.Shutdown(ctx))
		}
		if tp != nil {
			logger.Debug("Shutting down TracerProvider...")
			shutdownErr = errors.Join(shutdownErr, tp.Shutdown(ctx))
		}

		logger.Debug("Executing additional shutdown functions", zap.Int("count", len(shutdownFuncs)))
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		logger.Info("OpenTelemetry resources shut down sequence completed.")
		return shutdownErr
	}

	defer func() {
		if err != nil {
			logger.Error("OpenTelemetry SDK initialization failed", zap.Error(err))

			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
				logger.Error("Error during OTel cleanup after setup failure", zap.Error(shutdownErr))
			}

			manager.InitializeGlobalManager(nil, nil, nil, logger, cfg.ServiceName, cfg.ServiceVersion)
		} else {

			otel.SetTracerProvider(tp)
			otel.SetMeterProvider(mp)
			global.SetLoggerProvider(lp)

			manager.InitializeGlobalManager(tp, mp, lp, logger, cfg.ServiceName, cfg.ServiceVersion)
			logger.Info("OpenTelemetry SDK initialization completed successfully.")
		}
	}()

	res, err := resource.NewResource(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create resource: %w", err)
	}

	propagator.SetupPropagators()

	traceExporter, err := exporter.NewTraceExporter(ctx, cfg, logger)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	sampler := trace.NewSampler(cfg)

	var bspShutdown func(context.Context) error
	tp, bspShutdown = trace.NewTraceProvider(res, traceExporter, sampler)
	if bspShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, bspShutdown)
	}

	metricExporter, err := exporter.NewMetricExporter(ctx, cfg, logger)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp = metric.NewMeterProvider(cfg, res, metricExporter)

	logExporter, err := exporter.NewLogExporter(ctx, cfg, logger)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp, err = logotel.NewLoggerProvider(cfg, res, logExporter)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create logger provider: %w", err)
	}

	return shutdown, nil
}
