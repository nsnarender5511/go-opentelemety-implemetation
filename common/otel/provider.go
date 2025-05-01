package otel

import (
	"context"
	"time"

	"github.com/narender/common/config"
	"github.com/narender/common/lifecycle"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// ShutdownFunc is a function that handles graceful shutdown of a telemetry component
type ShutdownFunc func(ctx context.Context) error

// Setup encapsulates OpenTelemetry setup
type Setup struct {
	cfg            *config.Config
	logger         *logrus.Logger
	resource       *Resource
	tracerProvider *TracerProvider
	meterProvider  *MeterProvider
	loggerProvider *LoggerProvider
	shutdownFuncs  []ShutdownFunc
}

// Option is a function that configures a Setup
type Option func(*Setup)

// WithLogger sets the logger for the Setup
func WithLogger(logger *logrus.Logger) Option {
	return func(s *Setup) {
		s.logger = logger
	}
}

// NewSetup creates a new OpenTelemetry setup with the provided configuration and registers it with the shutdown manager
func NewSetup(ctx context.Context, cfg *config.Config, shutdownManager *lifecycle.ShutdownManager, opts ...Option) (*Setup, error) {
	// Create a setup with defaults
	s := &Setup{
		cfg:           cfg,
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

	// Register the main shutdown logic with the lifecycle manager
	// Use a reasonable timeout for OTel shutdown
	// TODO: Make this timeout configurable if needed
	const otelShutdownTimeout = 15 * time.Second
	if shutdownManager != nil {
		shutdownManager.Register("OpenTelemetry", s, otelShutdownTimeout)
		s.logger.Infof("Registered OpenTelemetry shutdown with lifecycle manager (timeout: %s)", otelShutdownTimeout)
	} else {
		s.logger.Warn("No lifecycle shutdown manager provided; OpenTelemetry shutdown will not be managed automatically.")
	}

	return s, nil
}

// WithResource creates and configures the OpenTelemetry resource
func (s *Setup) WithResource(ctx context.Context) (*Setup, error) {
	resource, err := newResource(ctx, s.cfg.ServiceName, s.cfg.ServiceVersion)
	if err != nil {
		return s, err
	}
	s.resource = resource
	s.logger.Debug("OpenTelemetry resource created")
	return s, nil
}

// WithPropagator sets up the OpenTelemetry propagator
func (s *Setup) WithPropagator() *Setup {
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
	s.logger.Debug("OpenTelemetry propagation configured")
	return s
}

// ensureResource ensures a resource exists, creating one if needed
func (s *Setup) ensureResource(ctx context.Context) error {
	if s.resource == nil {
		var err error
		resource, err := newResource(ctx, s.cfg.ServiceName, s.cfg.ServiceVersion)
		if err != nil {
			return err
		}
		s.resource = resource
		s.logger.Debug("OpenTelemetry resource created during ensureResource check")
	}
	return nil
}

// WithTracing sets up the OpenTelemetry tracer provider
func (s *Setup) WithTracing(ctx context.Context) (*Setup, error) {
	if err := s.ensureResource(ctx); err != nil {
		return s, err
	}

	tracerProvider, shutdown, err := newTracerProvider(ctx, s.cfg, s.resource, s.logger)
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

// WithMetrics sets up the OpenTelemetry meter provider
func (s *Setup) WithMetrics(ctx context.Context) (*Setup, error) {
	if err := s.ensureResource(ctx); err != nil {
		return s, err
	}

	meterProvider, shutdown, err := newMeterProvider(ctx, s.cfg, s.resource, s.logger)
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

// WithLogging sets up the OpenTelemetry logger provider
func (s *Setup) WithLogging(ctx context.Context) (*Setup, error) {
	if err := s.ensureResource(ctx); err != nil {
		return s, err
	}

	loggerProvider, shutdown, err := newLoggerProvider(ctx, s.cfg, s.resource, s.logger)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to initialize logger provider, proceeding without OpenTelemetry logging")
		return s, nil // Continue without logging
	}

	s.loggerProvider = loggerProvider
	s.addShutdownFunc(shutdown)

	// Configure logrus with OpenTelemetry
	configureLogrus(s.logger, loggerProvider, s.cfg.LogLevel)
	s.logger.Info("Logger provider configured")

	return s, nil
}

// addShutdownFunc adds a shutdown function to the list
func (s *Setup) addShutdownFunc(shutdown ShutdownFunc) {
	if shutdown != nil {
		s.shutdownFuncs = append(s.shutdownFuncs, shutdown)
	}
}

// Shutdown properly cleans up all telemetry resources
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
