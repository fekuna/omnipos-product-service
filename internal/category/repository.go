package category

import (
	"context"

	"github.com/fekuna/omnipos-product-service/internal/category/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, category *model.Category) error
	FindByID(ctx context.Context, id string) (*model.Category, error)
	FindAll(ctx context.Context, filters *dto.CategoryFilters) ([]model.Category, int, error)
	Update(ctx context.Context, category *model.Category) error
	Delete(ctx context.Context, id string) error
}
