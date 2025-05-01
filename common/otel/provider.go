package otel

import (
	"context"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type ShutdownFunc func(ctx context.Context) error

type Setup struct {
	logger         *logrus.Logger
	resource       *Resource
	tracerProvider *TracerProvider
	meterProvider  *MeterProvider
	loggerProvider *LoggerProvider
	shutdownFuncs  []ShutdownFunc
}

type Option func(*Setup)

func WithLogger(logger *logrus.Logger) Option {
	return func(s *Setup) {
		s.logger = logger
	}
}

func NewSetup(ctx context.Context, _ interface{}, opts ...Option) (*Setup, error) { // Changed shutdownManager type to ignore
	// Create a setup with defaults
	s := &Setup{
		logger:        logrus.StandardLogger(),
		shutdownFuncs: []ShutdownFunc{},
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Ensure resource exists before potentially setting up components that need it
	if err := s.ensureResource(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to ensure OpenTelemetry resource during initial setup")
		return nil, err
	}

	// Removed the block that registered shutdown with the lifecycle manager.
	// Manual shutdown via the returned Setup's Shutdown method is now required if graceful shutdown is needed elsewhere.
	s.logger.Warn("Automatic lifecycle manager registration removed; call Setup.Shutdown() manually if needed.")

	return s, nil
}

func (s *Setup) WithResource(ctx context.Context) (*Setup, error) {
	resource, err := newResource(ctx, config.SERVICE_NAME, config.SERVICE_VERSION)
	if err != nil {
		return s, err
	}
	s.resource = resource
	s.logger.Debug("OpenTelemetry resource created")
	return s, nil
}

func (s *Setup) WithPropagator() *Setup {
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
	s.logger.Debug("OpenTelemetry propagation configured")
	return s
}

func (s *Setup) ensureResource(ctx context.Context) error {
	if s.resource == nil {
		var err error
		resource, err := newResource(ctx, config.SERVICE_NAME, config.SERVICE_VERSION)
		if err != nil {
			return err
		}
		s.resource = resource
		s.logger.Debug("OpenTelemetry resource created during ensureResource check")
	}
	return nil
}

func (s *Setup) WithTracing(ctx context.Context) (*Setup, error) {
	if err := s.ensureResource(ctx); err != nil {
		return s, err
	}

	tracerProvider, shutdown, err := newTracerProvider(ctx, s.resource, s.logger)
	if err != nil {
		return s, err
	}

	s.tracerProvider = tracerProvider
	s.addShutdownFunc(shutdown)

	// Set as global provider
	otel.SetTracerProvider(tracerProvider)
	s.logger.Info("Trace provider registered globally")

	return s, nil
}

func (s *Setup) WithMetrics(ctx context.Context) (*Setup, error) {
	if err := s.ensureResource(ctx); err != nil {
		return s, err
	}

	meterProvider, shutdown, err := newMeterProvider(ctx, s.resource, s.logger)
	if err != nil {
		return s, err
	}

	s.meterProvider = meterProvider
	s.addShutdownFunc(shutdown)

	// Set as global provider
	otel.SetMeterProvider(meterProvider)
	s.logger.Info("Meter provider registered globally")

	return s, nil
}

func (s *Setup) WithLogging(ctx context.Context) (*Setup, error) {
	if err := s.ensureResource(ctx); err != nil {
		return s, err
	}

	loggerProvider, shutdown, err := newLoggerProvider(ctx, s.resource, s.logger)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to initialize logger provider, proceeding without OpenTelemetry logging")
		return s, nil // Continue without logging
	}

	s.loggerProvider = loggerProvider
	s.addShutdownFunc(shutdown)

	// Configure logrus with OpenTelemetry
	configureLogrus(s.logger, loggerProvider, config.LOG_LEVEL)
	s.logger.Info("Logger provider configured")

	return s, nil
}

func (s *Setup) addShutdownFunc(shutdown ShutdownFunc) {
	if shutdown != nil {
		s.shutdownFuncs = append(s.shutdownFuncs, shutdown)
	}
}

func (s *Setup) Shutdown(ctx context.Context) error {
	s.logger.Info("Starting telemetry shutdown")

	var lastErr error
	// Shutdown in reverse order (LIFO)
	for i := len(s.shutdownFuncs) - 1; i >= 0; i-- {
		if s.shutdownFuncs[i] == nil {
			continue
		}

		if err := s.shutdownFuncs[i](ctx); err != nil {
			s.logger.WithError(err).Errorf("Error during telemetry shutdown step %d", i)
			lastErr = err
		}
	}

	if lastErr != nil {
		s.logger.WithError(lastErr).Error("Telemetry shutdown completed with errors")
		return lastErr
	}

	s.logger.Info("Telemetry shutdown completed successfully")
	return nil
}
