package services

import (
	"log/slog"

	"context"

	apierrors "github.com/narender/common/apierrors"
	"github.com/narender/common/globals"
	"github.com/narender/product-service/src/models"
	"github.com/narender/product-service/src/repositories"
)

type ProductService interface {
	GetAll(ctx context.Context) ([]models.Product, *apierrors.AppError)
	GetByName(ctx context.Context, name string) (models.Product, *apierrors.AppError)
	UpdateStock(ctx context.Context, name string, newStock int) *apierrors.AppError
	GetByCategory(ctx context.Context, category string) ([]models.Product, *apierrors.AppError)
	BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError)
}

type productService struct {
	repo   repositories.ProductRepository
	logger *slog.Logger
}

func NewProductService(repo repositories.ProductRepository) ProductService {
	return &productService{
		repo:   repo,
		logger: globals.Logger(),
	}
}
