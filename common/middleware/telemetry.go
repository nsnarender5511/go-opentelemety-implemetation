package middleware

import (
	otelfiber "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
)

// OtelMiddleware wraps the standard otelfiber middleware.
// This provides a central point to potentially customize OTel middleware behavior,
// such as adding default span attributes or modifying the propagator.
// For now, it simply returns the default otelfiber middleware.
func OtelMiddleware(opts ...otelfiber.Option) fiber.Handler {
	// Example customization: Add default options if none provided
	// if len(opts) == 0 {
	// 	opts = append(opts, otelfiber.WithServerName("my-service"))
	// }
	return otelfiber.Middleware(opts...)
}
