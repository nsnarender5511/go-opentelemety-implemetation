package repositories

import (
	"log/slog"

	db "github.com/narender/common/db"
	"github.com/narender/common/globals"

	// Import common errors package
	"context"

	apierrors "github.com/narender/common/apierrors"
	"github.com/narender/master-store/src/models"
)

// Updated Interface - only includes GetAll since master-store now uses the product-service API
// for stock updates and doesn't need direct database access for individual products
type ProductRepository interface {
	GetAll(ctx context.Context) ([]models.Product, *apierrors.AppError)
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
