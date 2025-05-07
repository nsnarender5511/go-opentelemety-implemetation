package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	apierrors "github.com/narender/common/apierrors"
	"github.com/narender/common/globals"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// StockUpdateRequest represents the request body for updating stock
type StockUpdateRequest struct {
	Name  string `json:"name"`
	Stock int    `json:"stock"`
}

// ProductDetailsRequest represents the request body for getting product details
type ProductDetailsRequest struct {
	Name string `json:"name"`
}

// ProductDetails represents the product details returned from the product service
type ProductDetails struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}

// getProductFromProductService makes an HTTP request to get product details from the product-service
func getProductFromProductService(ctx context.Context, name string) (*ProductDetails, *apierrors.AppError) {
	logger := globals.Logger()

	newCtx, span := commontrace.StartSpan(ctx,
		attribute.String("product.name", name),
		attribute.String("remote_service", "product-service"),
		attribute.String("remote_operation", "get_product_details"),
	)
	ctx = newCtx
	defer func() {
		if span != nil {
			commontrace.EndSpan(span, nil, nil)
		}
	}()

	// Create request body
	reqBody := ProductDetailsRequest{
		Name: name,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.ErrorContext(ctx, "Shop Manager: Failed to create product details request to Warehouse",
			slog.String("error", err.Error()))

		if span != nil {
			span.SetStatus(codes.Error, "Failed to create JSON request")
		}

		return nil, apierrors.NewAppError(apierrors.ErrCodeInternal,
			"Failed to prepare product query for central warehouse", err)
	}

	// Get the product service URL from config
	productServiceURL := globals.Cfg().PRODUCT_SERVICE_URL
	url := fmt.Sprintf("%s/products/details", productServiceURL)

	logger.DebugContext(ctx, "Shop Manager: Requesting product details from Central Warehouse",
		slog.String("product", name),
		slog.String("warehouse_url", url))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.ErrorContext(ctx, "Shop Manager: Failed to create HTTP request for product details",
			slog.String("error", err.Error()))

		if span != nil {
			span.SetStatus(codes.Error, "Failed to create HTTP request")
		}

		return nil, apierrors.NewAppError(apierrors.ErrCodeInternal,
			"Failed to prepare network request to central warehouse", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client with auto-instrumentation
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   50 * time.Second, // Shorter timeout for read operations
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		logger.WarnContext(ctx, "Shop Manager: Failed to connect to Central Warehouse for product details",
			slog.String("error", err.Error()))

		if span != nil {
			span.SetStatus(codes.Error, "Failed to connect to product service")
		}

		return nil, apierrors.NewAppError(apierrors.ErrCodeServiceUnavailable,
			"Central Warehouse is currently unreachable for product information", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		logger.WarnContext(ctx, "Shop Manager: Central Warehouse couldn't provide product details",
			slog.Int("status_code", resp.StatusCode))

		if span != nil {
			span.SetStatus(codes.Error, "Product service returned non-OK status")
		}

		return nil, apierrors.NewAppError(apierrors.ErrCodeNotFound,
			"Product information unavailable from central warehouse",
			fmt.Errorf("product service returned status: %d", resp.StatusCode))
	}

	// Parse response
	var response struct {
		Data ProductDetails `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		logger.ErrorContext(ctx, "Shop Manager: Failed to understand product details from Central Warehouse",
			slog.String("error", err.Error()))

		if span != nil {
			span.SetStatus(codes.Error, "Failed to parse response")
		}

		return nil, apierrors.NewAppError(apierrors.ErrCodeInternal,
			"Failed to understand product information from central warehouse", err)
	}

	logger.DebugContext(ctx, "Shop Manager: Successfully retrieved product details from Central Warehouse",
		slog.String("product", name),
		slog.Float64("price", response.Data.Price),
		slog.String("category", response.Data.Category))

	return &response.Data, nil
}

// updateProductStockInProductService makes an HTTP request to the product-service to update stock
func updateProductStockInProductService(ctx context.Context, name string, newStock int) *apierrors.AppError {
	logger := globals.Logger()

	// Create request body
	reqBody := StockUpdateRequest{
		Name:  name,
		Stock: newStock,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.ErrorContext(ctx, "Shop Manager: Failed to create update request to Warehouse",
			slog.String("error", err.Error()))

		return apierrors.NewAppError(apierrors.ErrCodeInternal,
			"Failed to prepare update request to central warehouse", err)
	}

	// Get the product service URL from config
	productServiceURL := globals.Cfg().PRODUCT_SERVICE_URL
	url := fmt.Sprintf("%s/products/stock", productServiceURL)

	logger.InfoContext(ctx, "Shop Manager: Sending stock update request to Central Warehouse",
		slog.String("product", name),
		slog.Int("new_stock", newStock),
		slog.String("warehouse_url", url))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.ErrorContext(ctx, "Shop Manager: Failed to create HTTP request to Warehouse",
			slog.String("error", err.Error()))

		return apierrors.NewAppError(apierrors.ErrCodeInternal,
			"Failed to prepare network request to central warehouse", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client with auto-instrumentation
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   10 * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "Shop Manager: Failed to connect to Central Warehouse",
			slog.String("error", err.Error()))

		return apierrors.NewAppError(apierrors.ErrCodeServiceUnavailable,
			"Central Warehouse is currently unreachable, please try again later", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Central Warehouse rejected our stock update (status code: %d)", resp.StatusCode)
		logger.ErrorContext(ctx, "Shop Manager: Central Warehouse rejected our stock update",
			slog.Int("status_code", resp.StatusCode))

		return apierrors.NewAppError(apierrors.ErrCodeServiceUnavailable,
			errMsg, fmt.Errorf("product service returned status: %d", resp.StatusCode))
	}

	logger.InfoContext(ctx, "Shop Manager: Central Warehouse successfully updated their records",
		slog.String("product", name),
		slog.Int("new_stock", newStock))

	return nil
}
