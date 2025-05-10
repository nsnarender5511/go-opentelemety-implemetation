package repositories

import (
	"log/slog"

	db "github.com/narender/common/db"
	"github.com/narender/common/globals"

	// Import common errors package
	"context"

	apierrors "github.com/narender/common/apierrors"
	"github.com/narender/product-service/src/models"
)

// Updated Interface
type ProductRepository interface {
	GetAll(ctx context.Context) ([]models.Product, *apierrors.AppError)
	GetByName(ctx context.Context, name string) (models.Product, *apierrors.AppError)
	UpdateStock(ctx context.Context, name string, newStock int) *apierrors.AppError
	GetByCategory(ctx context.Context, category string) ([]models.Product, *apierrors.AppError)
}

type productRepository struct {
	database *db.FileDatabase
	logger   *slog.Logger
}

// NewProductRepository creates a new repository instance loading data from a JSON file.
func NewProductRepository() ProductRepository {
	repo := &productRepository{
		database: db.NewFileDatabase(),
		logger:   globals.Logger(),
	}
	return repo
}
