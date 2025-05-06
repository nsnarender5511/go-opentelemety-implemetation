package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	h.logger.DebugContext(c.UserContext(), "Shop open/close status requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
