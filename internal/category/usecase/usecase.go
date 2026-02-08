package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-product-service/internal/category"
	"github.com/fekuna/omnipos-product-service/internal/category/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/google/uuid"
)

type categoryUseCase struct {
	repo   category.Repository
	logger logger.ZapLogger
}

func NewCategoryUseCase(repo category.Repository, log logger.ZapLogger) category.UseCase {
	return &categoryUseCase{
		repo:   repo,
		logger: log,
	}
}

func (uc *categoryUseCase) CreateCategory(ctx context.Context, input *dto.CreateCategoryInput) (*model.Category, error) {
	// Validate parent if needed
	if input.ParentID != nil && *input.ParentID != "" {
		_, err := uc.repo.FindByID(ctx, *input.ParentID)
		if err != nil {
			return nil, err // Handle not found specifically?
		}
	}

	id := uuid.New().String()
	now := time.Now()

	cat := &model.Category{
		BaseModel: model.BaseModel{
			ID:        id,
			CreatedAt: now,
			UpdatedAt: now,
		},
		MerchantID:  input.MerchantID,
		ParentID:    input.ParentID,
		Name:        input.Name,
		Description: &input.Description,
		ImageURL:    &input.ImageURL,
		SortOrder:   input.SortOrder,
		IsActive:    true,
	}

	err := uc.repo.Create(ctx, cat)
	if err != nil {
		return nil, err
	}

	return cat, nil
}

func (uc *categoryUseCase) GetCategory(ctx context.Context, id string) (*model.Category, error) {
	return uc.repo.FindByID(ctx, id)
}

func (uc *categoryUseCase) ListCategories(ctx context.Context, filters *dto.CategoryFilters) ([]model.Category, int, error) {
	categories, count, err := uc.repo.FindAll(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	// If include_children, build tree?
	// Repo returns flat list. Tree building logic could be here strictly for frontend convenience,
	// but proto definition has 'children' field.
	// However, building full tree often requires fetching ALL categories or recursive queries.
	// For now, return flat list. The UI can build tree or we implement tree builder here.

	return categories, count, nil
}

func (uc *categoryUseCase) UpdateCategory(ctx context.Context, input *dto.UpdateCategoryInput) (*model.Category, error) {
	cat, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, errors.New("category not found")
	}

	// Update fields
	cat.Name = input.Name
	cat.Description = &input.Description
	cat.ImageURL = &input.ImageURL
	cat.SortOrder = input.SortOrder
	cat.IsActive = input.IsActive
	cat.ParentID = input.ParentID // Handle carefully logic for self-parenting loop check
	cat.UpdatedAt = time.Now()

	err = uc.repo.Update(ctx, cat)
	if err != nil {
		return nil, err
	}
	return cat, nil
}

func (uc *categoryUseCase) DeleteCategory(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}
