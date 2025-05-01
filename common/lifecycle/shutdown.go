package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
)

// Shutdowner defines an interface for components that support graceful shutdown
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// ShutdownManager handles the graceful shutdown of components
type ShutdownManager struct {
	logger     *logrus.Logger
	components []namedComponent
	timeout    time.Duration
	signalChan chan os.Signal
	stopChan   chan struct{}
	shutdownWg sync.WaitGroup
}

type namedComponent struct {
	name      string
	component Shutdowner
	timeout   time.Duration
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(logger *logrus.Logger) *ShutdownManager {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	return &ShutdownManager{
		logger:     logger,
		components: []namedComponent{},
		timeout:    30 * time.Second, // Default total timeout
		signalChan: make(chan os.Signal, 1),
		stopChan:   make(chan struct{}),
	}
}

// WithTimeout sets the total timeout for shutdown
func (m *ShutdownManager) WithTimeout(timeout time.Duration) *ShutdownManager {
	m.timeout = timeout
	return m
}

// Register adds a component to be shut down
func (m *ShutdownManager) Register(name string, component Shutdowner, timeout time.Duration) *ShutdownManager {
	m.components = append(m.components, namedComponent{
		name:      name,
		component: component,
		timeout:   timeout,
	})
	return m
}

// Start begins listening for shutdown signals
func (m *ShutdownManager) Start(ctx context.Context) {
	signal.Notify(m.signalChan, syscall.SIGINT, syscall.SIGTERM)

	m.shutdownWg.Add(1)
	go func() {
		defer m.shutdownWg.Done()

		select {
		case sig := <-m.signalChan:
			m.logger.WithField("signal", sig.String()).Info("Received shutdown signal")
			m.executeShutdown(ctx)
		case <-m.stopChan:
			m.logger.Info("Shutdown requested programmatically")
			m.executeShutdown(ctx)
		case <-ctx.Done():
			m.logger.Info("Shutdown triggered by context cancellation")
			m.executeShutdown(context.Background()) // Use a new context since the original is canceled
		}
	}()
}

// Stop triggers a shutdown programmatically
func (m *ShutdownManager) Stop() {
	select {
	case m.stopChan <- struct{}{}:
		// Signal sent
	default:
		// Channel already closed or shutdown in progress
	}

	// Wait for shutdown to complete
	m.shutdownWg.Wait()
}

// executeShutdown performs the actual shutdown sequence
func (m *ShutdownManager) executeShutdown(ctx context.Context) {
	// Create a context with timeout for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	var shutdownErrors []error

	// Shutdown components in reverse order (LIFO)
	for i := len(m.components) - 1; i >= 0; i-- {
		comp := m.components[i]

		// Create a context with the component's timeout
		compCtx, compCancel := context.WithTimeout(shutdownCtx, comp.timeout)

		m.logger.WithField("component", comp.name).Info("Shutting down component")
		startTime := time.Now()

		if err := comp.component.Shutdown(compCtx); err != nil {
			m.logger.WithError(err).WithField("component", comp.name).Error("Error shutting down component")
			shutdownErrors = append(shutdownErrors, fmt.Errorf("shutdown error for %s: %w", comp.name, err))
		} else {
			duration := time.Since(startTime)
			m.logger.WithFields(logrus.Fields{
				"component": comp.name,
				"duration":  duration,
			}).Info("Component shutdown successful")
		}

		compCancel()

		// Check if the overall context is done
		if shutdownCtx.Err() != nil {
			m.logger.Warn("Overall shutdown timeout exceeded, skipping remaining components")
			break
		}
	}

	if len(shutdownErrors) > 0 {
		m.logger.Error("Shutdown completed with errors")
		// Force exit with error code
		os.Exit(1)
	} else {
		m.logger.Info("Graceful shutdown completed successfully")
		os.Exit(0)
	}
}

// FiberAdapter adapts a Fiber app to the Shutdowner interface
type FiberAdapter struct {
	App *fiber.App
}

// Shutdown implements the Shutdowner interface
func (a *FiberAdapter) Shutdown(ctx context.Context) error {
	if a.App == nil {
		return nil
	}
	return a.App.ShutdownWithContext(ctx)
}

// WaitForGracefulShutdown provides backward compatibility with the previous API
func WaitForGracefulShutdown(ctx context.Context, server Shutdowner, telemetryShutdown func(context.Context) error) {
	cfg := config.GetConfig()
	logger := logrus.StandardLogger()

	// Create a shutdown manager
	manager := NewShutdownManager(logger).WithTimeout(cfg.ShutdownTotalTimeout)

	// Register components
	manager.Register("server", server, cfg.ShutdownServerTimeout)

	// Adapt the telemetry shutdown function to a Shutdowner
	if telemetryShutdown != nil {
		telemetryComp := &functionAdapter{fn: telemetryShutdown}
		manager.Register("telemetry", telemetryComp, cfg.ShutdownOtelMinTimeout)
	}

	// Start the manager and wait for signals
	manager.Start(ctx)

	// Block until shutdown completes
	select {} // This will block until the process exits
}

// functionAdapter adapts a function to the Shutdowner interface
type functionAdapter struct {
	fn func(context.Context) error
}

// Shutdown calls the wrapped function
func (a *functionAdapter) Shutdown(ctx context.Context) error {
	if a.fn == nil {
		return nil
	}
	return a.fn(ctx)
}
