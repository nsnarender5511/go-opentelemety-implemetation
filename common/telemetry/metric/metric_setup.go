package metric

import (
	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
)

func NewMeterProvider(cfg *config.Config, res *resource.Resource, exporter sdkmetric.Exporter) *sdkmetric.MeterProvider {
	log := manager.GetLogger()
	mpOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),

		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithTimeout(cfg.OtelExporterOtlpTimeout))),
	}

	if cfg.OtelEnableExemplars {
		log.Info("Enabling OpenTelemetry Exemplars for metrics using TraceBasedFilter.")

		mpOpts = append(mpOpts, sdkmetric.WithExemplarFilter(exemplar.TraceBasedFilter))
	} else {
		log.Info("OpenTelemetry Exemplars are disabled.")
	}

	mp := sdkmetric.NewMeterProvider(mpOpts...)
	log.Info("Meter provider configured.")
	return mp
}
