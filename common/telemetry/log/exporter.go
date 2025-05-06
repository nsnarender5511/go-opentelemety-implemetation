package log

import (
	"context"
	"fmt"
	"log"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	logger "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

func SetupOtlpLogExporter(ctx context.Context, cfg *config.Config, connOpts []grpc.DialOption, res *sdkresource.Resource) error {
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OTEL_ENDPOINT),
		otlploggrpc.WithDialOption(connOpts...),
		otlploggrpc.WithInsecure(),
	)
	fmt.Println("OTEL_ENDPOINT :: ", cfg.OTEL_ENDPOINT)
	if err != nil {
		return fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	logProcessor := sdklog.NewBatchProcessor(logExporter)
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(logProcessor),
	)
	logger.SetLoggerProvider(loggerProvider)
	log.Println("OTel LoggerProvider initialized and set globally.")
	return nil
}
