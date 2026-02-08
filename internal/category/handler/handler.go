package handler

import (
	"context"

	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-product-service/internal/auth"
	"github.com/fekuna/omnipos-product-service/internal/category"
	"github.com/fekuna/omnipos-product-service/internal/category/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
	pb "github.com/fekuna/omnipos-proto/gen/go/omnipos/product/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ pb.CategoryServiceServer = (*CategoryHandler)(nil)

type CategoryHandler struct {
	pb.UnimplementedCategoryServiceServer
	uc     category.UseCase
	logger logger.ZapLogger
}

func NewCategoryHandler(uc category.UseCase, log logger.ZapLogger) *CategoryHandler {
	return &CategoryHandler{
		uc:     uc,
		logger: log,
	}
}

func (h *CategoryHandler) CreateCategory(ctx context.Context, req *pb.CreateCategoryRequest) (*pb.CreateCategoryResponse, error) {
	merchantID := auth.GetMerchantID(ctx)
	if merchantID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing merchant context")
	}

	input := &dto.CreateCategoryInput{
		MerchantID:  merchantID,
		ParentID:    nil,
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageUrl,
		SortOrder:   int(req.SortOrder),
	}
	if req.ParentId != "" {
		input.ParentID = &req.ParentId
	}

	cat, err := h.uc.CreateCategory(ctx, input)
	if err != nil {
		h.logger.Error("failed to create category", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CreateCategoryResponse{
		Category: mapModelToProto(cat),
	}, nil
}

func (h *CategoryHandler) GetCategory(ctx context.Context, req *pb.GetCategoryRequest) (*pb.GetCategoryResponse, error) {
	cat, err := h.uc.GetCategory(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if cat == nil {
		return nil, status.Error(codes.NotFound, "category not found")
	}

	// Ensure merchant ownership check if needed (usually Repo filters by merchant, but GetCategory input has only ID)
	// If IDs are UUIDs, collision unlikely, but strict multi-tenancy requires checking MerchantID.
	merchantID := auth.GetMerchantID(ctx)
	if cat.MerchantID != merchantID {
		return nil, status.Error(codes.NotFound, "category not found")
	}

	return &pb.GetCategoryResponse{
		Category: mapModelToProto(cat),
	}, nil
}

func (h *CategoryHandler) ListCategories(ctx context.Context, req *pb.ListCategoriesRequest) (*pb.ListCategoriesResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	// Map filters
	filters := &dto.CategoryFilters{
		MerchantID:      merchantID,
		IsActive:        nil,
		IncludeChildren: req.IncludeChildren,
	}
	if req.ParentId != "" {
		filters.ParentID = &req.ParentId
	}
	// Note: proto ListCategoriesRequest definition for 'is_active' is boolean, default false.
	// This makes it hard to filter "all" vs "inactive". Often proto needs wrapper or assumption.
	// Assuming if req.IsActive is true, we filter active only. If false, we might return all or active?
	// Let's assume frontend passes true for active only.
	if req.IsActive {
		active := true
		filters.IsActive = &active
	}

	cats, count, err := h.uc.ListCategories(ctx, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoCats := make([]*pb.Category, len(cats))
	for i, c := range cats {
		protoCats[i] = mapModelToProto(&c)
	}

	return &pb.ListCategoriesResponse{
		Categories: protoCats,
		Total:      int32(count),
	}, nil
}

func (h *CategoryHandler) UpdateCategory(ctx context.Context, req *pb.UpdateCategoryRequest) (*pb.UpdateCategoryResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	input := &dto.UpdateCategoryInput{
		ID:          req.Id,
		MerchantID:  merchantID,
		Name:        req.Name,
		Description: req.Description,
		ImageURL:    req.ImageUrl,
		SortOrder:   int(req.SortOrder),
		IsActive:    req.IsActive,
	}
	if req.ParentId != "" {
		input.ParentID = &req.ParentId
	}

	cat, err := h.uc.UpdateCategory(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateCategoryResponse{
		Category: mapModelToProto(cat),
	}, nil
}

func (h *CategoryHandler) DeleteCategory(ctx context.Context, req *pb.DeleteCategoryRequest) (*emptypb.Empty, error) {
	// TODO: Verify ownership
	err := h.uc.DeleteCategory(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// Helper to map model to proto
func mapModelToProto(m *model.Category) *pb.Category {
	if m == nil {
		return nil
	}

	// Handle Children if any
	var children []*pb.Category
	if len(m.Children) > 0 {
		children = make([]*pb.Category, len(m.Children))
		for i, c := range m.Children {
			children[i] = mapModelToProto(&c)
		}
	}

	parentID := ""
	if m.ParentID != nil {
		parentID = *m.ParentID
	}

	desc := ""
	if m.Description != nil {
		desc = *m.Description
	}

	imgURL := ""
	if m.ImageURL != nil {
		imgURL = *m.ImageURL
	}

	return &pb.Category{
		Id:          m.ID,
		MerchantId:  m.MerchantID,
		ParentId:    parentID,
		Name:        m.Name,
		Description: desc,
		ImageUrl:    imgURL,
		SortOrder:   int32(m.SortOrder),
		IsActive:    m.IsActive,
		CreatedAt:   timestamppb.New(m.CreatedAt),
		UpdatedAt:   timestamppb.New(m.UpdatedAt),
		Children:    children,
	}
}
