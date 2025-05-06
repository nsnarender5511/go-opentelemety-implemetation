package metric

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

func SetupOtlpMetricExporter(ctx context.Context, cfg *config.Config, connOpts []grpc.DialOption, res *sdkresource.Resource) error {
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTEL_ENDPOINT),
		otlpmetricgrpc.WithDialOption(connOpts...),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)
	log.Println("OTel MeterProvider initialized and set globally.")
	return nil
}
