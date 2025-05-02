package middleware

import (
	"errors"
	"fmt"
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
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

func NewErrorHandler(logger *logrus.Logger, metrics *otel.Metrics) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		statusCode := commonErrors.ToStatusCode(err)

		resp := ErrorResponse{
			StatusCode: statusCode,
			Message:    "An unexpected error occurred. Please try again later.",
		}

		var appErr *commonErrors.AppError
		if errors.As(err, &appErr) {
			if appErr.UserMessage != "" {
				resp.Message = appErr.UserMessage
			} else if appErr.Message != "" {
				resp.Message = appErr.Message
			}
		} else {
		}

		span := oteltrace.SpanFromContext(c.UserContext())
		if span != nil && span.IsRecording() {
			otel.RecordSpanError(span, err, otel.AttrHTTPResponseStatusCodeKey.Int(statusCode))
		}

		if metrics != nil {
			attrs := []attribute.KeyValue{
				otel.AttrHTTPRequestMethod.String(c.Method()),
				otel.AttrHTTPRouteKey.String(c.Route().Path),
				otel.AttrHTTPResponseStatus.Int(statusCode),
			}

			metrics.RecordHTTPRequestDuration(c.UserContext(), 0*time.Second /* duration unknown here */, attrs...)
		}

		entry := logger.WithFields(logrus.Fields{
			"error":       err.Error(),
			"status_code": statusCode,
			"method":      c.Method(),
			"path":        c.Path(),
			"ip":          c.IP(),
			"route":       c.Route().Path,
		})
		if span != nil && span.SpanContext().IsValid() {
			entry = entry.WithFields(logrus.Fields{
				"trace_id": span.SpanContext().TraceID().String(),
				"span_id":  span.SpanContext().SpanID().String(),
			})
		}

		if statusCode >= 500 {
			entry.Error(fmt.Sprintf("Server error occurred: %s", err.Error()))
		} else {
			entry.Warn(fmt.Sprintf("Client error occurred: %s", err.Error()))
		}

		c.Status(statusCode)
		return c.JSON(resp)
	}
}
