package category

import (
	"context"

	"github.com/fekuna/omnipos-product-service/internal/category/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
)

type UseCase interface {
	CreateCategory(ctx context.Context, input *dto.CreateCategoryInput) (*model.Category, error)
	GetCategory(ctx context.Context, id string) (*model.Category, error)
	ListCategories(ctx context.Context, filters *dto.CategoryFilters) ([]model.Category, int, error)
	UpdateCategory(ctx context.Context, input *dto.UpdateCategoryInput) (*model.Category, error)
	DeleteCategory(ctx context.Context, id string) error
}
