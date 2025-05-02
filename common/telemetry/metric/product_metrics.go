package metric

import (
	"fmt"

	"github.com/narender/common/telemetry/manager"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	ProductInstrumentationName = "github.com/narender/product-service"
	productStock               = "product.stock"
)





func DefineProductStockGauge() (otelmetric.Int64ObservableGauge, error) {
	meter := manager.GetMeter(ProductInstrumentationName)
	logger := manager.GetLogger()

	stockGauge, err := meter.Int64ObservableGauge(
		productStock,
		otelmetric.WithDescription("Current number of products in stock"),
		otelmetric.WithUnit("{items}"),
	)
	if err != nil {
		logger.Error("Failed to create product.stock gauge", zap.Error(err))
		return nil, fmt.Errorf("failed to create %s gauge: %w", productStock, err)
	}
	logger.Info("Defined observable gauge", zap.String("gauge_name", productStock))
	return stockGauge, nil
}
