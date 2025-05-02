package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	_ "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/logging"
	"github.com/narender/common/telemetry/attributes"
	"github.com/narender/common/telemetry/metric"
	"github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ErrorResponse struct {
	StatusCode int                    `json:"statusCode"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func NewErrorHandler(baseLogger *zap.Logger, metrics *metric.HTTPMetrics) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		statusCode := http.StatusInternalServerError
		userMessage := "An unexpected error occurred. Please try again later."
		internalMessage := err.Error()
		var details map[string]interface{}
		errorType := commonErrors.TypeUnknown
		logLevel := zapcore.ErrorLevel
		var appErr *commonErrors.AppError
		var fiberErr *fiber.Error
		var validationErr *commonErrors.ValidationError
		var dbErr *commonErrors.DatabaseError
		if errors.As(err, &appErr) {

			if appErr.StatusCode != 0 {
				statusCode = appErr.StatusCode
			}
			if appErr.UserMessage != "" {
				userMessage = appErr.UserMessage
			}
			internalMessage = appErr.Error()
			details = appErr.Context
			errorType = appErr.Type
		} else if errors.As(err, &fiberErr) {

			statusCode = fiberErr.Code
			userMessage = fiberErr.Message
			internalMessage = fiberErr.Error()
		} else if errors.As(err, &validationErr) {

			statusCode = http.StatusBadRequest
			userMessage = validationErr.Error()
			internalMessage = validationErr.Error()
			errorType = commonErrors.TypeValidation
		} else if errors.As(err, &dbErr) {

			statusCode = http.StatusInternalServerError
			userMessage = "A database error occurred."
			internalMessage = dbErr.Error()
			errorType = commonErrors.TypeDatabase
		} else {

			if errors.Is(err, commonErrors.ErrNotFound) {
				statusCode = http.StatusNotFound
				userMessage = "Resource not found."
				errorType = commonErrors.TypeNotFound
			} else if errors.Is(err, commonErrors.ErrInvalidInput) || errors.Is(err, commonErrors.ErrBadRequest) {
				statusCode = http.StatusBadRequest
				userMessage = "Invalid request."
				errorType = commonErrors.TypeBadRequest
			}
		}

		if statusCode >= 400 && statusCode < 500 {
			logLevel = zapcore.WarnLevel
		} else if statusCode >= 500 {
			logLevel = zapcore.ErrorLevel
		}

		ctx := c.UserContext()
		span := oteltrace.SpanFromContext(ctx)
		if span != nil && span.IsRecording() {
			trace.RecordSpanError(span, err)
			span.SetAttributes(
				attribute.String("error.message", internalMessage),
				attribute.Int("error.type", int(errorType)),
			)
		}

		if metrics != nil {
			attrs := []attribute.KeyValue{
				attributes.HTTPMethodKey.String(c.Method()),
				attributes.HTTPRouteKey.String(c.Route().Path),
				attributes.HTTPStatusCodeKey.Int(statusCode),
			}

			metrics.RecordHTTPRequestDuration(ctx, 0*time.Second, attrs...)
		}

		logger := logging.LoggerFromContext(ctx)

		zapFields := []zap.Field{
			zap.Error(err),
			zap.String("internal_message", internalMessage),
			zap.Int("error_type", int(errorType)),
			zap.Int("status_code", statusCode),
			zap.String("user_message", userMessage),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("route", c.Route().Path),
			zap.Any("request_details", details),
		}

		logMessage := fmt.Sprintf("HTTP Error: %s %s -> %d", c.Method(), c.Path(), statusCode)

		if ce := logger.Check(logLevel, logMessage); ce != nil {
			ce.Write(zapFields...)
		}

		resp := ErrorResponse{
			StatusCode: statusCode,
			Message:    userMessage,
			Details:    details,
		}
		c.Status(statusCode)
		return c.JSON(resp)
	}
}
