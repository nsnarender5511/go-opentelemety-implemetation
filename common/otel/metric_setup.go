package otel

import (
	"github.com/narender/common/config"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/resource"
)


func newMeterProvider(cfg *config.Config, res *resource.Resource, exporter sdkmetric.Exporter) *sdkmetric.MeterProvider {
	log := GetLogger()
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
