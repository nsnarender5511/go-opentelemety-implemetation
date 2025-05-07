package handlers

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func (h *MasterStoreHandler) HealthCheck(c *fiber.Ctx) error {
	h.logger.DebugContext(c.UserContext(), "Master store health check requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
