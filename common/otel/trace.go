package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Common Semantic Attribute Keys for Tracing
// See: https://opentelemetry.io/docs/specs/semconv/general/trace/
// And specific conventions (HTTP, DB, RPC, etc.)
var (
	// General
	AttrServiceName    = semconv.ServiceNameKey
	AttrServiceVersion = semconv.ServiceVersionKey
	AttrDeploymentEnv  = semconv.DeploymentEnvironmentKey

	// HTTP Client/Server (subset)
	AttrHTTPRequestMethodKey      = semconv.HTTPRequestMethodKey
	AttrHTTPResponseStatusCodeKey = semconv.HTTPResponseStatusCodeKey
	AttrHTTPSchemeKey             = semconv.URLSchemeKey
	AttrHTTPTargetKey             = semconv.URLPathKey       // Or semconv.HTTPTargetKey depending on context
	AttrHTTPRouteKey              = semconv.HTTPRouteKey     // The matched route (path template)
	AttrNetPeerIPKey              = semconv.ClientAddressKey // Or semconv.NetPeerNameKey / semconv.ServerSocketAddressKey
	AttrNetHostNameKey            = semconv.NetHostNameKey
	AttrNetHostPortKey            = semconv.NetHostPortKey
	AttrUserAgentOriginalKey      = semconv.UserAgentOriginalKey

	// Database Client (subset)
	AttrDBSystemKey    = semconv.DBSystemKey    // e.g., "postgresql", "mysql", "redis"
	AttrDBNameKey      = semconv.DBNameKey      // Database name
	AttrDBStatementKey = semconv.DBStatementKey // The query text
	AttrDBOperationKey = semconv.DBOperationKey // e.g., "SELECT", "INSERT"

	// Messaging (subset)
	AttrMessagingSystemKey          = semconv.MessagingSystemKey          // e.g., "kafka", "rabbitmq"
	AttrMessagingDestinationNameKey = semconv.MessagingDestinationNameKey // e.g., topic or queue name
	AttrMessagingOperationKey       = semconv.MessagingOperationKey       // e.g., "publish", "receive"
	AttrMessagingMessageIDKey       = semconv.MessagingMessageIDKey

	// Error Attributes
	AttrExceptionTypeKey       = semconv.ExceptionTypeKey
	AttrExceptionMessageKey    = semconv.ExceptionMessageKey
	AttrExceptionStacktraceKey = semconv.ExceptionStacktraceKey
)

// RecordSpanError sets the span status to Error, records the error event with stacktrace,
// and adds standard error attributes.
// It accepts additional attributes specific to the error context.
func RecordSpanError(span oteltrace.Span, err error, attributes ...attribute.KeyValue) {
	if span == nil || err == nil || !span.IsRecording() {
		return
	}

	// Set span status to Error
	span.SetStatus(codes.Error, err.Error()) // Use error message for description

	// Prepare attributes for the event
	eventAttrs := []attribute.KeyValue{
		AttrExceptionTypeKey.String(fmt.Sprintf("%T", err)), // Record the type of the error
		AttrExceptionMessageKey.String(err.Error()),
		// AttrExceptionStacktraceKey.String(...), // OTel Go SDK often automatically captures stacktrace with RecordError
	}
	eventAttrs = append(eventAttrs, attributes...) // Add any custom attributes

	// Record the error event on the span.
	// The SDK's RecordError implementation often includes the stack trace automatically.
	span.RecordError(err, oteltrace.WithAttributes(eventAttrs...))
}
