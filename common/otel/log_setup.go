package otel

import (
	"github.com/narender/common/config"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newLoggerProvider(cfg *config.Config, res *resource.Resource, exporter sdklog.Exporter) (*sdklog.LoggerProvider, error) {

	var processorOpts []sdklog.BatchProcessorOption
	if cfg.OtelBatchTimeout > 0 {
		processorOpts = append(processorOpts, sdklog.WithExportTimeout(cfg.OtelBatchTimeout))
	}

	proc := sdklog.NewBatchProcessor(exporter, processorOpts...)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(proc),
	)
	GetLogger().Info("Logger provider configured.")
	return lp, nil
}
