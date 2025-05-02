package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/logging" 
	"go.uber.org/zap"                    
)




func RequestLoggerMiddleware() fiber.Handler { 
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		
		
		logger := logging.LoggerFromContext(c.UserContext())

		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		
		zapFields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status_code", statusCode),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.String("ip", c.IP()),
		}

		
		if err != nil {
			zapFields = append(zapFields, zap.Error(err))
		}

		
		if statusCode >= 500 {
			logger.Error("Request completed with server error", zapFields...)
		} else if statusCode >= 400 {
			logger.Warn("Request completed with client error", zapFields...)
		} else {
			logger.Info("Request completed successfully", zapFields...)
		}

		
		return err
	}
}
