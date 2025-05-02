package middleware

import (
	otelfiber "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
)

func OtelMiddleware(opts ...otelfiber.Option) fiber.Handler {
	return otelfiber.Middleware(opts...)
}
