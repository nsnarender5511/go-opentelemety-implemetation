package metric

// import (
// 	"github.com/gofiber/fiber/v2/log"
// 	"github.com/narender/common/config"
// 	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
// 	"go.opentelemetry.io/otel/sdk/metric/exemplar"
// 	"go.opentelemetry.io/otel/sdk/resource"
// )

// func NewMeterProvider(cfg *config.Config, res *resource.Resource, exporter sdkmetric.Exporter) *sdkmetric.MeterProvider {
// 	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.OtelBatchTimeout))

// 	mpOpts := []sdkmetric.Option{
// 		sdkmetric.WithResource(res),
// 		sdkmetric.WithReader(reader),
// 	}

// 	if cfg.OtelEnableExemplars {
// 		log.Info("Enabling OpenTelemetry Exemplars for metrics using TraceBasedFilter.")
// 		mpOpts = append(mpOpts, sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter))
// 	} else {
// 		log.Info("OpenTelemetry Exemplars are disabled.")
// 	}

// 	provider := sdkmetric.NewMeterProvider(mpOpts...)
// 	log.Info("Meter provider configured.")
// 	return provider
// }
