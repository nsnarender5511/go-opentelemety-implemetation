package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
)

// WaitForGracefulShutdown blocks until a SIGINT or SIGTERM signal is received,
// then coordinates the graceful shutdown of the provided server and telemetry.
// It uses timeout configurations from the common/config package.
func WaitForGracefulShutdown(ctx context.Context, server Shutdowner, telemetryShutdown func(context.Context) error) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	// Use the globally configured logrus logger
	logger := logrus.StandardLogger()

	logger.WithField("signal", sig.String()).Info("Received shutdown signal, initiating graceful shutdown...")

	// Use background context for shutdown process, independent of initial context
	shutdownTotalTimeout := config.ShutdownTotalTimeout()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTotalTimeout)
	defer cancel()

	var shutdownErrs error
	// Define shutdown order and specific timeouts from config
	shutdownTasks := []struct {
		name     string
		timeout  time.Duration // Specific timeout for this task
		shutdown func(context.Context) error
	}{
		{"server", config.ShutdownServerTimeout(), server.Shutdown},
		{"telemetry", config.ShutdownOtelMinTimeout(), telemetryShutdown},
	}

	// Process shutdowns sequentially
	for _, task := range shutdownTasks {
		if task.shutdown == nil {
			logger.Debugf("Skipping shutdown for %s (nil function)", task.name)
			continue
		}

		// Create context with specific timeout for this task, derived from the overall shutdown context.
		// This ensures the task doesn't exceed its allocated time AND respects the overall deadline.
		taskCtx, taskCancel := context.WithTimeout(shutdownCtx, task.timeout)

		logger.Infof("Attempting to shut down %s (timeout: %s)...", task.name, task.timeout)
		if err := task.shutdown(taskCtx); err != nil {
			logger.WithError(err).Errorf("Error during %s shutdown", task.name)
			shutdownErrs = errors.Join(shutdownErrs, fmt.Errorf("%s shutdown error: %w", task.name, err))
			// If the error is context deadline exceeded, log it specifically
			if errors.Is(err, context.DeadlineExceeded) {
				logger.Warnf("%s shutdown timed out after %s", task.name, task.timeout)
			}
		} else {
			logger.Infof("%s shutdown complete", task.name)
		}
		taskCancel() // Cancel this task's context immediately

		// Check if the *overall* shutdown context has timed out after this step
		if shutdownCtx.Err() != nil {
			logger.Warnf("Overall shutdown timeout (%s) exceeded during %s shutdown. Aborting further steps.", shutdownTotalTimeout, task.name)
			// Ensure the timeout error is captured if not already
			if !errors.Is(shutdownErrs, context.DeadlineExceeded) {
				shutdownErrs = errors.Join(shutdownErrs, fmt.Errorf("overall shutdown timeout exceeded: %w", shutdownCtx.Err()))
			}
			break // Stop processing further shutdown tasks
		}
	}

	if shutdownErrs != nil {
		logger.WithError(shutdownErrs).Error("Application shutdown completed with errors")
		os.Exit(1) // Exit with error code if any shutdown step failed
	} else {
		logger.Info("Application shutdown completed successfully")
		os.Exit(0) // Exit normally
	}
}
