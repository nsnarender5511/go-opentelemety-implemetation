package lifecycle

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// Shutdowner defines an interface for components that support graceful shutdown.
// This allows the common shutdown helper to work with different server types
// (e.g., Fiber, Gin, net/http) that provide a compatible Shutdown method.
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// FiberShutdownAdapter adapts a *fiber.App to the Shutdowner interface.
// It uses Fiber's ShutdownWithContext for preferred graceful shutdown.
type FiberShutdownAdapter struct {
	App *fiber.App
}

// Shutdown calls ShutdownWithContext on the wrapped Fiber app.
func (a *FiberShutdownAdapter) Shutdown(ctx context.Context) error {
	if a.App == nil {
		return nil // Or return an error if app shouldn't be nil
	}
	return a.App.ShutdownWithContext(ctx)
}
