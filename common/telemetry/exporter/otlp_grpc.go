package exporter

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager" // Keep manager import for logger
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// newOTLPGrpcConnection remains unexported as it's internal to the exporter package.
func newOTLPGrpcConnection(ctx context.Context, cfg *config.Config, signalType string) (*grpc.ClientConn, error) {
	logger := manager.GetLogger()
	var transportCreds credentials.TransportCredentials
	if cfg.OtelExporterInsecure {
		transportCreds = insecure.NewCredentials()
		logger.Warnf("Using insecure gRPC connection for OTLP %s exporter", signalType)
	} else {
		logger.Infof("Using secure gRPC connection for OTLP %s exporter", signalType)
		transportCreds = credentials.NewTLS(&tls.Config{})
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
	}

	dialCtx, cancel := context.WithTimeout(ctx, cfg.OtelExporterOtlpTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, cfg.OtelExporterOtlpEndpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial OTLP %s exporter endpoint %s: %w", signalType, cfg.OtelExporterOtlpEndpoint, err)
	}
	logger.Infof("Successfully connected to OTLP gRPC endpoint for %s: %s", signalType, cfg.OtelExporterOtlpEndpoint)
	return conn, nil
}
