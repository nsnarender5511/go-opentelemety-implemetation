package telemetry

// TelemetryConfig contains configuration for OpenTelemetry instrumentation.
// This struct provides a centralized way to configure all aspects of telemetry
// including tracing, metrics, and logging.
type TelemetryConfig struct {
	// ServiceName identifies this service in telemetry data
	ServiceName string

	// Endpoint is the OTLP endpoint URL (e.g., "localhost:4317")
	Endpoint string

	// Headers for OTLP exporter (e.g., for authentication)
	Headers map[string]string

	// Insecure disables TLS for the OTLP exporter
	Insecure bool

	// SampleRatio controls the trace sampling rate (0.0-1.0)
	// 1.0 means sample all traces, 0.0 means sample none
	SampleRatio float64

	// BatchTimeoutMS for the span processor
	// Controls how long to wait before sending a batch of spans
	BatchTimeoutMS int

	// MaxExportBatchSize controls max spans per batch
	// Recommended values: 512-1024
	MaxExportBatchSize int

	// LogLevel specifies the minimum log level (e.g., "info", "debug")
	LogLevel string

	// LogFormat specifies the log output format (e.g., "text", "json")
	LogFormat string
}
