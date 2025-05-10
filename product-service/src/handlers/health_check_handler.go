package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Get request ID

	h.logger.DebugContext(ctx, "Health check requested",
		slog.String("component", "product_handler"),
		slog.String("operation", "health_check"))

	// Create response with request ID
	response := fiber.Map{
		"status": "ok",
	}

	return c.Status(http.StatusOK).JSON(response)
}
