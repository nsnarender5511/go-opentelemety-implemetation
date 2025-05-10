package telemetry

import (
	"context"
	"fmt"
	"log"

	"github.com/narender/common/config"
	logExporter "github.com/narender/common/telemetry/log"
	metricExporter "github.com/narender/common/telemetry/metric"
	otelemetryResource "github.com/narender/common/telemetry/resource"
	traceExporter "github.com/narender/common/telemetry/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitTelemetry(cfg *config.Config) error {

	res, err := otelemetryResource.NewResource(context.Background(), cfg.SERVICE_NAME, cfg.SERVICE_VERSION)
	if err != nil {

		log.Printf("ERROR: Failed to create OTel resource: %v\n", err)
		return fmt.Errorf("failed to create resource: %w", err)
	}
	log.Println("OTel Resource created.")

	if cfg.ENVIRONMENT == "production" {
		log.Println("Production environment detected. Initializing OTLP Trace, Metric, and Log providers.")

		ctx := context.Background()
		connOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}

		if err := traceExporter.SetupOtlpTraceExporter(ctx, cfg, connOpts, res); err != nil {
			log.Printf("ERROR: OTLP Trace exporter setup failed: %v\n", err)
			return fmt.Errorf("trace exporter setup failed: %w", err)
		}

		if err := metricExporter.SetupOtlpMetricExporter(ctx, cfg, connOpts, res); err != nil {
			log.Printf("ERROR: OTLP Metric exporter setup failed: %v\n", err)
			return fmt.Errorf("metric exporter setup failed: %w", err)
		}

		if err := logExporter.SetupOtlpLogExporter(ctx, cfg, connOpts, res); err != nil {
			log.Printf("ERROR: OTLP Log exporter setup failed: %v\n", err)
			return fmt.Errorf("log exporter setup failed: %w", err)
		}

	} else {

		log.Printf("Non-production environment (%s) detected. Skipping OTLP exporter setup. Using No-Op providers.", cfg.ENVIRONMENT)

	}

	log.Println("OpenTelemetry SDK initialization sequence complete.")
	return nil
}
