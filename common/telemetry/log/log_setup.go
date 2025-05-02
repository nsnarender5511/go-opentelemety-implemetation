package log

import (
	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func NewLoggerProvider(cfg *config.Config, res *resource.Resource, exporter sdklog.Exporter) (*sdklog.LoggerProvider, error) {

	var processorOpts []sdklog.BatchProcessorOption
	if cfg.OtelBatchTimeout > 0 {
		processorOpts = append(processorOpts, sdklog.WithExportTimeout(cfg.OtelBatchTimeout))
	}

	proc := sdklog.NewBatchProcessor(exporter, processorOpts...)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(proc),
	)
	manager.GetLogger().Info("Logger provider configured.")
	return lp, nil
}
