package telemetry

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sirupsen/logrus" // Your logging library

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc" // Corrected path
	"go.opentelemetry.io/otel/log/global"                         // Global logger provider

	// Import the OTel Log API package
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource" // Added for attributes

	// To extract trace context
	"google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure" // Ensure this line is commented out or removed
)

var (
	// Global instance to hold the configured OTel LoggerProvider
	otelLoggerProvider *sdklog.LoggerProvider
	// Mutex to protect access to the global provider during configuration
	providerMu sync.Mutex
)

// initLoggerProvider initializes the OTLP Logger Provider.
// It sets up the exporter and the provider but doesn't set it globally yet.
func initLoggerProvider(ctx context.Context, endpoint string, insecure bool, res *resource.Resource) (shutdownFunc func(context.Context) error, err error) {
	// --- Create OTLP Exporter ---
	exporterOpts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
		// otlploggrpc.WithCompressor(grpc.UseCompressor(gzip.Name)),
		// otlploggrpc.WithHeaders(map[string]string{"api-key": "your-key"}),
		otlploggrpc.WithDialOption(grpc.WithBlock()),
	}

	if insecure {
		exporterOpts = append(exporterOpts, otlploggrpc.WithInsecure())
	} else {
		// Use secure transport (TLS)
		log.Println("Attempting to use secure OTLP log exporter.")
		// No explicit credentials option means default secure gRPC
	}

	logExporter, err := otlploggrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}
	log.Println("OTLP Log Exporter created.")

	// --- Create Batch Log Record Processor ---
	blrp := sdklog.NewBatchProcessor(logExporter)
	log.Println("Batch Log Record Processor created.")

	// --- Create Logger Provider ---
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(blrp),
	)
	log.Println("Logger Provider created.")

	// --- Store the provider globally (protected by mutex) ---
	providerMu.Lock()
	otelLoggerProvider = loggerProvider
	providerMu.Unlock()
	// Set the global delegate provider (needed for logrus hook to work)
	global.SetLoggerProvider(loggerProvider)
	log.Println("Global Logger Provider delegate set.")

	// Return the shutdown function.
	shutdown := func(shutdownCtx context.Context) error {
		log.Println("Shutting down Logger Provider...")
		providerMu.Lock()
		lp := otelLoggerProvider // Get the provider under lock
		otelLoggerProvider = nil // Reset global reference
		providerMu.Unlock()

		if lp == nil {
			log.Println("Logger Provider already shut down or not initialized.")
			return nil
		}

		// Use a timeout for the shutdown context.
		ctx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := lp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down Logger Provider: %v", err)
			return fmt.Errorf("failed to shutdown LoggerProvider: %w", err)
		}
		log.Println("Logger Provider shut down successfully.")
		return nil
	}

	return shutdown, nil
}

// --- Custom Logrus Hook for OpenTelemetry ---

// OtelHook implements logrus.Hook to send logs to OpenTelemetry.
type OtelHook struct {
	// Levels defines on which log levels this hook should fire.
	LogLevels []logrus.Level
}

// NewOtelHook creates a new hook for the specified levels.
func NewOtelHook(levels []logrus.Level) *OtelHook {
	return &OtelHook{LogLevels: levels}
}

// Levels returns the log levels this hook is registered for.
func (h *OtelHook) Levels() []logrus.Level {
	if h.LogLevels == nil {
		return logrus.AllLevels // Default to all levels if not specified
	}
	return h.LogLevels
}

// Fire is called by Logrus when a log entry is made.
func (h *OtelHook) Fire(entry *logrus.Entry) error {
	providerMu.Lock()
	lp := otelLoggerProvider // Get current provider safely
	providerMu.Unlock()

	if lp == nil {
		// Provider not yet initialized or already shut down, do nothing.
		return nil
	}

	// Get a logger instance from the global provider.
	otelLogger := lp.Logger(
		"logrus", // Instrumentation scope name
	)

	// Prepare attributes using otellog.KeyValue
	attrs := make([]otellog.KeyValue, 0, len(entry.Data)+5) // Change type here
	for k, v := range entry.Data {
		attrs = append(attrs, otelAttributeFromInterface(k, v)) // This now returns otellog.KeyValue
	}
	if entry.HasCaller() {
		// Use otellog constructors for caller info as well
		attrs = append(attrs, otellog.String("logrus.code.function", entry.Caller.Function))
		attrs = append(attrs, otellog.String("logrus.code.filepath", entry.Caller.File))
		attrs = append(attrs, otellog.Int("logrus.code.lineno", entry.Caller.Line))
	}

	// Get context (potentially containing trace/span info)
	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Construct the otellog.Record (API record)
	apiRecord := otellog.Record{}
	apiRecord.SetTimestamp(entry.Time)
	apiRecord.SetObservedTimestamp(time.Now())
	apiRecord.SetSeverity(mapSeverity(entry.Level))
	apiRecord.SetBody(otellog.StringValue(entry.Message))
	// Add collected attributes to the record
	apiRecord.AddAttributes(attrs...)

	// Emit the API record using the logger instance, passing context
	// Trace/Span context is implicitly carried by ctx
	otelLogger.Emit(ctx, apiRecord)

	return nil
}

// configureLoggerProvider sets up logging with OTel (if needed in the future)
// For now, it just returns a stub function
func configureLoggerProvider(ctx context.Context, config TelemetryConfig, res *resource.Resource) (func(context.Context) error, error) {
	logger := config.Logger
	if logger == nil {
		logger = getLogger()
	}

	logger.Debug("Logger provider configuration is minimal in this implementation")

	// Return a no-op shutdown function
	shutdown := func(context.Context) error {
		logger.Debug("Logger provider shutdown (no-op)")
		return nil
	}

	return shutdown, nil
}

// ConfigureLogrus sets up logrus with the specified level and formatter
// and adds an OpenTelemetry hook if available
func ConfigureLogrus(logger *logrus.Logger, level, format string) {
	// Parse log level with fallback to info
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
		logger.WithFields(logrus.Fields{
			"provided_level": level,
			"fallback_level": "info",
		}).Warn("Invalid log level specified, using info level")
	}

	logger.SetLevel(logLevel)

	// Configure formatter based on format
	switch format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:    true,
			TimestampFormat:  "2006-01-02T15:04:05.000Z07:00",
			DisableColors:    false,
			DisableTimestamp: false,
			PadLevelText:     true,
		})
	}

	// Check if OTel Logger Provider is available and add hook
	providerMu.Lock()
	initialized := otelLoggerProvider != nil
	providerMu.Unlock()

	if initialized {
		hook := NewOtelHook(logrus.AllLevels)
		logger.AddHook(hook)
		logger.Debug("OpenTelemetry logrus hook configured")
	}
}

// --- Helper Functions ---

// mapSeverity maps logrus log levels to OpenTelemetry severity levels.
func mapSeverity(level logrus.Level) otellog.Severity {
	switch level {
	case logrus.TraceLevel:
		return otellog.SeverityTrace
	case logrus.DebugLevel:
		return otellog.SeverityDebug
	case logrus.InfoLevel:
		return otellog.SeverityInfo
	case logrus.WarnLevel:
		return otellog.SeverityWarn
	case logrus.ErrorLevel:
		return otellog.SeverityError
	case logrus.FatalLevel:
		return otellog.SeverityFatal
	case logrus.PanicLevel:
		return otellog.SeverityFatal
	default:
		return otellog.SeverityInfo
	}
}

// otelAttributeFromInterface converts a key-value pair to an OTel KeyValue attribute.
func otelAttributeFromInterface(key string, value interface{}) otellog.KeyValue {
	switch v := value.(type) {
	case string:
		return otellog.String(key, v)
	case int:
		return otellog.Int(key, v)
	case int64:
		return otellog.Int64(key, v)
	case float64:
		return otellog.Float64(key, v)
	case bool:
		return otellog.Bool(key, v)
	default:
		// Convert anything else to string
		return otellog.String(key, fmt.Sprintf("%v", v))
	}
}
