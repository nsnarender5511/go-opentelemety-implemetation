package logging

import (
	"context"
	"fmt"
	"time"

	log "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global" // Import for the global LoggerProvider
	"go.opentelemetry.io/otel/trace"      // Import trace package

	"github.com/sirupsen/logrus"
)

// OtelHook implements logrus.Hook to add trace context and emit OTel logs.
type OtelHook struct {
	// We don't need to store the OTel logger explicitly if we use the global one.
	// otelLogger log.Logger
}

// NewOtelHook creates a new hook instance.
// It assumes the global OTel LoggerProvider has been set.
func NewOtelHook() *OtelHook {
	// Optionally, get a specific logger instance here if needed,
	// but using the global provider is often sufficient.
	// provider := global.GetLoggerProvider()
	// logger := provider.Logger("logrus-otel-hook") // Or a more specific name
	return &OtelHook{
		// otelLogger: logger,
	}
}

// Levels returns the log levels that this hook should fire for.
func (h *OtelHook) Levels() []logrus.Level {
	// Fire for all levels handled by Logrus
	return logrus.AllLevels
}

// Fire executes the hook logic for a given log entry.
func (h *OtelHook) Fire(entry *logrus.Entry) error {
	// 1. Extract context if available
	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background() // Use background context if none provided
	}

	// 2. Get span context
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	// 3. Add trace/span IDs to Logrus fields if valid
	if spanCtx.IsValid() {
		entry.Data["trace_id"] = spanCtx.TraceID().String()
		entry.Data["span_id"] = spanCtx.SpanID().String()
	}

	// 4. Emit an OTel Log Record using the global LoggerProvider
	// Get the logger from the global provider. You might want to specify an instrumentation scope.
	otelLogger := global.GetLoggerProvider().Logger(
		"github.com/narender/common/logging/otel_hook", // Instrumentation scope name
		// Optionally add schema URL, version, etc.
		// log.WithInstrumentationVersion("1.0.0"),
		// log.WithSchemaURL(semconv.SchemaURL),
	)

	// Prepare OTel log record
	record := log.Record{}
	record.SetTimestamp(entry.Time) // Use Logrus timestamp
	record.SetObservedTimestamp(time.Now())
	record.SetSeverity(mapLogLevel(entry.Level))
	record.SetSeverityText(entry.Level.String())
	record.SetBody(log.StringValue(entry.Message))

	// Add attributes from Logrus fields + trace/span IDs
	for k, v := range entry.Data {
		// Convert common types to OTel attributes
		// This might need refinement based on expected data types in Logrus fields
		switch val := v.(type) {
		case string:
			record.AddAttributes(log.String(k, val))
		case int:
			record.AddAttributes(log.Int(k, val))
		case int64:
			record.AddAttributes(log.Int64(k, val))
		case float64:
			record.AddAttributes(log.Float64(k, val))
		case bool:
			record.AddAttributes(log.Bool(k, val))
		case error: // Include error message if present
			record.AddAttributes(log.String(k, val.Error()))
		default: // Fallback for other types
			record.AddAttributes(log.String(k, fmt.Sprintf("%+v", val)))
		}
	}

	// Emit the record using the OTel logger
	otelLogger.Emit(ctx, record) // Pass the original context

	return nil
}

// mapLogLevel converts Logrus level to OTel severity number.
func mapLogLevel(level logrus.Level) log.Severity {
	switch level {
	case logrus.TraceLevel:
		return log.SeverityTrace
	case logrus.DebugLevel:
		return log.SeverityDebug
	case logrus.InfoLevel:
		return log.SeverityInfo
	case logrus.WarnLevel:
		return log.SeverityWarn
	case logrus.ErrorLevel:
		return log.SeverityError
	case logrus.FatalLevel:
		return log.SeverityFatal
	case logrus.PanicLevel:
		return log.SeverityFatal // OTel doesn't have a direct Panic level
	default:
		return log.SeverityInfo // Default fallback
	}
}
