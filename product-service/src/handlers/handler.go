package handlers

import (
	"log/slog"

	"github.com/narender/common/globals"
	"github.com/narender/product-service/src/services"
)

type ProductHandler struct {
	service services.ProductService // Adjusted to use services.ProductService
	logger  *slog.Logger
}

func NewProductHandler(svc services.ProductService) *ProductHandler { // Adjusted to use services.ProductService
	return &ProductHandler{
		service: svc,
		logger:  globals.Logger(),
	}
}
