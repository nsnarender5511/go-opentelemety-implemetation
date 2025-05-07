package services

import (
	"log/slog"

	"context"

	apierrors "github.com/narender/common/apierrors"
	"github.com/narender/common/globals"
	"github.com/narender/master-store/src/models"
	"github.com/narender/master-store/src/repositories"
)

type MasterStoreService interface {
	GetAll(ctx context.Context) ([]models.Product, *apierrors.AppError)
	BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError)
	UpdateStock(ctx context.Context, name string, newStock int) *apierrors.AppError
}

type masterStoreService struct {
	repo   repositories.ProductRepository
	logger *slog.Logger
}

func NewMasterStoreService(repo repositories.ProductRepository) MasterStoreService {
	return &masterStoreService{
		repo:   repo,
		logger: globals.Logger(),
	}
}
