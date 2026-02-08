package handler

import (
	"context"

	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-product-service/internal/auth"
	"github.com/fekuna/omnipos-product-service/internal/model"
	"github.com/fekuna/omnipos-product-service/internal/product"
	"github.com/fekuna/omnipos-product-service/internal/product/dto"
	productv1 "github.com/fekuna/omnipos-proto/proto/product/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductHandler struct {
	productv1.UnimplementedProductServiceServer
	productv1.UnimplementedProductVariantServiceServer // Can verify if this interface is generated separately? Yes it is defined as separate service in proto.
	// BUT wait, my product.proto defined TWO services: ProductService and ProductVariantService.
	// I should implement both or have separate handlers.
	// I'll implement both in this file for simplicity as they share the UseCase.

	uc     product.UseCase
	logger logger.ZapLogger
}

func NewProductHandler(uc product.UseCase, log logger.ZapLogger) *ProductHandler {
	return &ProductHandler{
		uc:     uc,
		logger: log,
	}
}

// --- ProductService Server ---

func (h *ProductHandler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.ProductResponse, error) {
	merchantID := auth.GetMerchantID(ctx)
	if merchantID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing merchant")
	}

	input := &dto.CreateProductInput{
		MerchantID:     merchantID,
		CategoryID:     req.CategoryId,
		SKU:            req.Sku,
		Barcode:        req.Barcode,
		Name:           req.Name,
		Description:    req.Description,
		BasePrice:      req.BasePrice,
		CostPrice:      req.CostPrice,
		TaxRate:        req.TaxRate,
		HasVariants:    req.HasVariants,
		TrackInventory: req.TrackInventory,
		ImageURL:       req.ImageUrl,
	}

	p, err := h.uc.CreateProduct(ctx, input)
	if err != nil {
		h.logger.Error("failed to create product", zap.Error(err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &productv1.ProductResponse{
		Product: mapProductToProto(p),
	}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.ProductResponse, error) {
	p, err := h.uc.GetProduct(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if p == nil {
		return nil, status.Error(codes.NotFound, "product not found")
	}

	return &productv1.ProductResponse{Product: mapProductToProto(p)}, nil
}

func (h *ProductHandler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	isActive := (*bool)(nil)
	if req.IsActive {
		b := true
		isActive = &b
	}

	filters := &dto.ProductFilters{
		MerchantID: merchantID,
		CategoryID: req.CategoryId,
		IsActive:   isActive,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       int(req.Page),
		PageSize:   int(req.PageSize),
	}

	products, count, err := h.uc.ListProducts(ctx, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protos := make([]*productv1.Product, len(products))
	for i, p := range products {
		protos[i] = mapProductToProto(&p)
	}

	return &productv1.ListProductsResponse{
		Products: protos,
		Total:    int32(count),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (h *ProductHandler) SearchProducts(ctx context.Context, req *productv1.SearchProductsRequest) (*productv1.ListProductsResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	filters := &dto.ProductFilters{
		MerchantID:  merchantID,
		SearchQuery: req.Query,
		Page:        1,
		PageSize:    int(req.Limit),
	}

	products, count, err := h.uc.ListProducts(ctx, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protos := make([]*productv1.Product, len(products))
	for i, p := range products {
		protos[i] = mapProductToProto(&p)
	}

	return &productv1.ListProductsResponse{
		Products: protos,
		Total:    int32(count),
	}, nil
}

func (h *ProductHandler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.ProductResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	input := &dto.UpdateProductInput{
		ID:             req.Id,
		MerchantID:     merchantID,
		CategoryID:     req.CategoryId,
		SKU:            req.Sku,
		Barcode:        req.Barcode,
		Name:           req.Name,
		Description:    req.Description,
		BasePrice:      req.BasePrice,
		CostPrice:      req.CostPrice,
		TaxRate:        req.TaxRate,
		TrackInventory: req.TrackInventory,
		ImageURL:       req.ImageUrl,
		IsActive:       req.IsActive,
	}

	p, err := h.uc.UpdateProduct(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &productv1.ProductResponse{Product: mapProductToProto(p)}, nil
}

func (h *ProductHandler) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*emptypb.Empty, error) {
	err := h.uc.DeleteProduct(ctx, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// Helper
func mapProductToProto(m *model.Product) *productv1.Product {
	if m == nil {
		return nil
	}

	catID := ""
	if m.CategoryID != nil {
		catID = *m.CategoryID
	}

	barcode := ""
	if m.Barcode != nil {
		barcode = *m.Barcode
	}

	costPrice := 0.0
	if m.CostPrice != nil {
		costPrice = *m.CostPrice
	}

	desc := ""
	if m.Description != nil {
		desc = *m.Description
	}

	imgURL := ""
	if m.ImageURL != nil {
		imgURL = *m.ImageURL
	}

	return &productv1.Product{
		Id:             m.ID,
		MerchantId:     m.MerchantID,
		CategoryId:     catID,
		Sku:            m.SKU,
		Barcode:        barcode,
		Name:           m.Name,
		Description:    desc,
		BasePrice:      m.BasePrice,
		CostPrice:      costPrice,
		TaxRate:        m.TaxRate,
		HasVariants:    m.HasVariants,
		TrackInventory: m.TrackInventory,
		ImageUrl:       imgURL,
		IsActive:       m.IsActive,
		CreatedAt:      timestamppb.New(m.CreatedAt),
		UpdatedAt:      timestamppb.New(m.UpdatedAt),
		// Variants and Category would be populated if JOINed or filled
	}
}

func (h *ProductHandler) ReserveStock(ctx context.Context, req *productv1.ReserveStockRequest) (*productv1.ReserveStockResponse, error) {
	merchantID := auth.GetMerchantID(ctx)
	if merchantID == "" {
		return nil, status.Error(codes.Unauthenticated, "missing merchant context")
	}

	items := make(map[string]int32)
	for _, item := range req.Items {
		// If variant_id is present, use it? Or both?
		// Implementation Plan said "Product Service: Implement Atomic DB Reservation... UPDATE products...".
		// It mentions treating variants later.
		// For now, key is ProductID.
		if item.ProductId == "" {
			continue
		}
		items[item.ProductId] = item.Quantity
	}

	if len(items) == 0 {
		return &productv1.ReserveStockResponse{
			Success: false,
			Message: "No valid items to reserve",
		}, nil
	}

	err := h.uc.ReserveStock(ctx, items)
	if err != nil {
		// Log warning rather than error for business logic failures?
		// Or return success=false.
		// If error is "insufficient stock", return success=false
		return &productv1.ReserveStockResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &productv1.ReserveStockResponse{
		Success: true,
	}, nil
}
