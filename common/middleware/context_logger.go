package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/logging"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)




func ContextLoggerMiddleware(baseLogger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.UserContext()      
		requestLogger := baseLogger 

		
		span := trace.SpanFromContext(ctx)
		spanCtx := span.SpanContext()

		if spanCtx.IsValid() {
			
			requestLogger = baseLogger.With(
				zap.String("trace_id", spanCtx.TraceID().String()),
				zap.String("span_id", spanCtx.SpanID().String()),
			)
		}

		
		
		newCtx := logging.NewContextWithLogger(ctx, requestLogger)

		
		
		c.SetUserContext(newCtx)

		
		return c.Next()
	}
}
