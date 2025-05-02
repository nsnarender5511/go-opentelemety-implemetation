package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	_ "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ErrorResponse struct {
	StatusCode int                    `json:"statusCode"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func NewErrorHandler(logger *logrus.Logger, metrics *otel.HTTPMetrics) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		statusCode := http.StatusInternalServerError
		userMessage := "An unexpected error occurred. Please try again later."
		internalMessage := err.Error()
		var details map[string]interface{}
		errorType := commonErrors.TypeUnknown
		logLevel := logrus.ErrorLevel
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
			logLevel = logrus.WarnLevel
		} else if statusCode >= 500 {
			logLevel = logrus.ErrorLevel
		}

		span := oteltrace.SpanFromContext(c.UserContext())
		if span != nil && span.IsRecording() {
			otel.RecordSpanError(span, err, otel.HTTPStatusCodeKey.Int(statusCode))
		}

		if metrics != nil {
			attrs := []attribute.KeyValue{
				otel.HTTPMethodKey.String(c.Method()),
				otel.HTTPRouteKey.String(c.Route().Path),
				otel.HTTPStatusCodeKey.Int(statusCode),
			}

			metrics.RecordHTTPRequestDuration(c.UserContext(), 0*time.Second, attrs...)
		}

		entry := logger.WithFields(logrus.Fields{
			"error":           internalMessage,
			"error_type":      fmt.Sprintf("%d", errorType),
			"status_code":     statusCode,
			"user_message":    userMessage,
			"method":          c.Method(),
			"path":            c.Path(),
			"ip":              c.IP(),
			"route":           c.Route().Path,
			"request_details": details,
		})
		if span != nil && span.SpanContext().IsValid() {
			entry = entry.WithFields(logrus.Fields{
				"trace_id": span.SpanContext().TraceID().String(),
				"span_id":  span.SpanContext().SpanID().String(),
			})
		}
		entry.Log(logLevel, fmt.Sprintf("%s %s resulted in %d: %s", c.Method(), c.Path(), statusCode, internalMessage))

		resp := ErrorResponse{
			StatusCode: statusCode,
			Message:    userMessage,
			Details:    details,
		}
		c.Status(statusCode)
		return c.JSON(resp)
	}
}
