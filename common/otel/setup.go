package otel

import (
	"context"
	"errors"
	"fmt"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Initialize the OpenTelemetry SDK with tracing, metrics, and logging configured.
func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {

	logger := SetupLogrus(cfg)

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

		logger.Debugf("Executing %d additional shutdown functions...", len(shutdownFuncs))
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		logger.Info("OpenTelemetry resources shut down sequence completed.")
		return shutdownErr
	}

	defer func() {
		if err != nil {
			logger.WithError(err).Error("OpenTelemetry SDK initialization failed")

			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
				logger.WithError(shutdownErr).Error("Error during OTel cleanup after setup failure")
			}

			initializeGlobalManager(nil, nil, nil, logger, cfg.ServiceName, cfg.ServiceVersion)
		} else {

			otel.SetTracerProvider(tp)
			otel.SetMeterProvider(mp)
			global.SetLoggerProvider(lp)

			initializeGlobalManager(tp, mp, lp, logger, cfg.ServiceName, cfg.ServiceVersion)
			logger.Info("OpenTelemetry SDK initialization completed successfully.")
		}
	}()

	res, err := newResource(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create resource: %w", err)
	}

	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)
	logger.Debug("Global TextMapPropagator configured.")

	traceExporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	sampler := newSampler(cfg)

	var bspShutdown func(context.Context) error
	tp, bspShutdown = newTraceProvider(res, traceExporter, sampler)
	if bspShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, bspShutdown)
	}

	metricExporter, err := newMetricExporter(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp = newMeterProvider(cfg, res, metricExporter)

	logExporter, err := newLogExporter(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp, err = newLoggerProvider(cfg, res, logExporter)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create logger provider: %w", err)
	}

	return shutdown, nil
}
