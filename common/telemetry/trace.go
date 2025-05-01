package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// newTraceProvider creates and configures a new TraceProvider
func newTraceProvider(ctx context.Context, config TelemetryConfig, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var err error

	logger := config.Logger
	if logger == nil {
		logger = getLogger()
	}

	logger.WithFields(logrus.Fields{
		"endpoint": config.Endpoint,
		"insecure": config.Insecure,
	}).Debug("Creating trace exporter")

	// Configure security options
	var secureOption otlptracegrpc.Option
	if config.Insecure {
		secureOption = otlptracegrpc.WithInsecure()
		logger.Debug("Using insecure connection for trace exporter")
	} else {
		// Use TLS credentials
		creds := credentials.NewClientTLSFromCert(nil, "")
		secureOption = otlptracegrpc.WithTLSCredentials(creds)
		logger.Debug("Using secure connection for trace exporter")
	}

	// Configure headers
	var headers map[string]string
	if config.Headers != nil {
		headers = config.Headers
	} else {
		headers = make(map[string]string)
	}

	// Create OTLP trace exporter
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(config.Endpoint),
		secureOption,
		otlptracegrpc.WithHeaders(headers),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Configure batch timeout
	batchTimeout := time.Duration(config.BatchTimeoutMS) * time.Millisecond
	if batchTimeout == 0 {
		batchTimeout = 5000 * time.Millisecond // Default 5 seconds
	}

	// Configure batch size
	batchSize := config.MaxExportBatchSize
	if batchSize == 0 {
		batchSize = 512 // Default batch size
	}

	// Create batch span processor
	bsp := sdktrace.NewBatchSpanProcessor(
		exp,
		sdktrace.WithBatchTimeout(batchTimeout),
		sdktrace.WithMaxExportBatchSize(batchSize),
	)

	// Configure sampler based on sample ratio
	var sampler sdktrace.Sampler
	if config.SampleRatio >= 1.0 {
		logger.Debug("Using AlwaysSample sampler")
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRatio <= 0.0 {
		logger.Debug("Using NeverSample sampler")
		sampler = sdktrace.NeverSample()
	} else {
		logger.WithField("ratio", config.SampleRatio).Debug("Using TraceIDRatioBased sampler")
		sampler = sdktrace.TraceIDRatioBased(config.SampleRatio)
	}

	// Create and return tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	logger.Info("Trace provider initialized successfully")
	return tp, nil
}

// GetTracer returns a named tracer instance from the global provider
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
