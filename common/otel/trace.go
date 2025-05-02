package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	AttrServiceName    = semconv.ServiceNameKey
	AttrServiceVersion = semconv.ServiceVersionKey
	AttrDeploymentEnv  = semconv.DeploymentEnvironmentKey

	AttrHTTPRequestMethodKey      = semconv.HTTPRequestMethodKey
	AttrHTTPResponseStatusCodeKey = semconv.HTTPResponseStatusCodeKey
	AttrHTTPSchemeKey             = semconv.URLSchemeKey
	AttrHTTPTargetKey             = semconv.URLPathKey
	AttrHTTPRouteKey              = semconv.HTTPRouteKey
	AttrNetPeerIPKey              = semconv.ClientAddressKey
	AttrNetHostNameKey            = semconv.NetHostNameKey
	AttrNetHostPortKey            = semconv.NetHostPortKey
	AttrUserAgentOriginalKey      = semconv.UserAgentOriginalKey

	AttrDBSystemKey    = semconv.DBSystemKey
	AttrDBNameKey      = semconv.DBNameKey
	AttrDBStatementKey = semconv.DBStatementKey
	AttrDBOperationKey = semconv.DBOperationKey

	AttrMessagingSystemKey          = semconv.MessagingSystemKey
	AttrMessagingDestinationNameKey = semconv.MessagingDestinationNameKey
	AttrMessagingOperationKey       = semconv.MessagingOperationKey
	AttrMessagingMessageIDKey       = semconv.MessagingMessageIDKey

	AttrExceptionTypeKey       = semconv.ExceptionTypeKey
	AttrExceptionMessageKey    = semconv.ExceptionMessageKey
	AttrExceptionStacktraceKey = semconv.ExceptionStacktraceKey
)

func RecordSpanError(span oteltrace.Span, err error, attributes ...attribute.KeyValue) {
	if span == nil || err == nil || !span.IsRecording() {
		return
	}

	span.SetStatus(codes.Error, err.Error())

	eventAttrs := []attribute.KeyValue{
		AttrExceptionTypeKey.String(fmt.Sprintf("%T", err)),
		AttrExceptionMessageKey.String(err.Error()),
	}
	eventAttrs = append(eventAttrs, attributes...)

	span.RecordError(err, oteltrace.WithAttributes(eventAttrs...))
}
