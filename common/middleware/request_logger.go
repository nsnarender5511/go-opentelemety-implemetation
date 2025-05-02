package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	commonlog "github.com/narender/common/log"
	"go.uber.org/zap"
)

func RequestLoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		logger := commonlog.L.Ctx(c.UserContext())

		zapFields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status_code", statusCode),
			zap.Duration("duration", duration),
			zap.String("ip", c.IP()),
			zap.String("user_agent", string(c.Request().Header.UserAgent())),
		}

		if err != nil {
			zapFields = append(zapFields, zap.Error(err))
		}

		msg := "Request completed"
		if statusCode >= 500 {
			logger.Error(msg, zapFields...)
		} else if statusCode >= 400 {
			logger.Warn(msg, zapFields...)
		} else {
			logger.Info(msg, zapFields...)
		}

		return err
	}
}
