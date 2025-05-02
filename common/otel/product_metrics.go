package otel

import (
	"fmt"

	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	ProductInstrumentationName = "github.com/narender/product-service"
	productStock               = "product.stock"
)

/*
var (
	AttrAppProductIDKey = semconv.AppProductIDKey
)
*/

// DefineProductStockGauge defines the observable gauge for product stock.
// It returns the instrument so the application can register its own callback.
func DefineProductStockGauge() (otelmetric.Int64ObservableGauge, error) {
	meter := GetMeter(ProductInstrumentationName)
	logger := GetLogger()

	stockGauge, err := meter.Int64ObservableGauge(
		productStock,
		otelmetric.WithDescription("Current number of products in stock"),
		otelmetric.WithUnit("{items}"),
	)
	if err != nil {
		logger.WithError(err).Error("Failed to create product.stock gauge")
		return nil, fmt.Errorf("failed to create %s gauge: %w", productStock, err)
	}
	logger.Infof("Defined %s observable gauge.", productStock)
	return stockGauge, nil
}
