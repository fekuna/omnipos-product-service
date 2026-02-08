package handler

import (
	"context"

	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-product-service/internal/auth"
	"github.com/fekuna/omnipos-product-service/internal/inventory"
	"github.com/fekuna/omnipos-product-service/internal/inventory/dto"
	"github.com/fekuna/omnipos-product-service/internal/model"
	productv1 "github.com/fekuna/omnipos-proto/gen/go/omnipos/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InventoryHandler struct {
	productv1.UnimplementedInventoryServiceServer
	uc     inventory.UseCase
	logger logger.ZapLogger
}

func NewInventoryHandler(uc inventory.UseCase, log logger.ZapLogger) *InventoryHandler {
	return &InventoryHandler{
		uc:     uc,
		logger: log,
	}
}

func (h *InventoryHandler) GetProductInventory(ctx context.Context, req *productv1.GetProductInventoryRequest) (*productv1.ProductInventoryResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	storeID := (*string)(nil)
	if req.StoreId != "" {
		s := req.StoreId
		storeID = &s
	}

	inv, err := h.uc.GetProductInventory(ctx, merchantID, req.ProductId, storeID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &productv1.ProductInventoryResponse{
		Inventory: []*productv1.InventoryEntry{mapInventoryToProto(inv)},
	}, nil
}

func (h *InventoryHandler) ListLowStock(ctx context.Context, req *productv1.ListLowStockRequest) (*productv1.ListLowStockResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	storeID := (*string)(nil)
	if req.StoreId != "" {
		s := req.StoreId
		storeID = &s
	}

	items, count, err := h.uc.ListLowStock(ctx, merchantID, storeID, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	entries := make([]*productv1.InventoryEntry, len(items))
	for i, item := range items {
		entries[i] = mapInventoryToProto(&item)
	}

	return &productv1.ListLowStockResponse{
		Items: entries,
		Total: int32(count),
	}, nil
}

func (h *InventoryHandler) AdjustInventory(ctx context.Context, req *productv1.AdjustInventoryRequest) (*productv1.InventoryEntry, error) {
	merchantID := auth.GetMerchantID(ctx)

	userID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if val := md.Get("x-user-id"); len(val) > 0 {
			userID = val[0]
		}
	}

	storeID := (*string)(nil)
	if req.StoreId != "" {
		s := req.StoreId
		storeID = &s
	}

	variantID := (*string)(nil)
	if req.VariantId != "" {
		v := req.VariantId
		variantID = &v
	}

	input := &dto.AdjustInventoryInput{
		MerchantID:     merchantID,
		StoreID:        storeID,
		ProductID:      req.ProductId,
		VariantID:      variantID,
		QuantityChange: req.QuantityChange,
		Reason:         req.Reason,
		ReferenceID:    req.ReferenceId,
		ReferenceType:  "manual", // Simplified
		UserID:         userID,
	}

	inv, err := h.uc.AdjustInventory(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return mapInventoryToProto(inv), nil
}

func (h *InventoryHandler) TransferInventory(ctx context.Context, req *productv1.TransferInventoryRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Transfer not implemented yet")
}

func (h *InventoryHandler) ListInventoryMovements(ctx context.Context, req *productv1.ListInventoryMovementsRequest) (*productv1.ListInventoryMovementsResponse, error) {
	merchantID := auth.GetMerchantID(ctx)

	sID := (*string)(nil)
	if req.StoreId != "" {
		s := req.StoreId
		sID = &s
	}

	filters := &dto.MovementFilters{
		MerchantID:   merchantID,
		ProductID:    req.ProductId,
		StoreID:      sID,
		MovementType: req.MovementType,
		Page:         int(req.Page),
		PageSize:     int(req.PageSize),
	}

	mvs, count, err := h.uc.ListMovements(ctx, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoMovements := make([]*productv1.InventoryMovement, len(mvs))
	for i, m := range mvs {
		protoMovements[i] = mapMovementToProto(&m)
	}

	return &productv1.ListInventoryMovementsResponse{
		Movements: protoMovements,
		Total:     int32(count),
	}, nil
}

func mapInventoryToProto(m *model.Inventory) *productv1.InventoryEntry {
	if m == nil {
		return nil
	}

	storeID := ""
	if m.StoreID != nil {
		storeID = *m.StoreID
	}
	variantID := ""
	if m.VariantID != nil {
		variantID = *m.VariantID
	}
	lastCounted := timestamppb.New(m.UpdatedAt) // fallback
	if m.LastCountedAt != nil {
		lastCounted = timestamppb.New(*m.LastCountedAt)
	}

	return &productv1.InventoryEntry{
		Id:                m.ID,
		MerchantId:        m.MerchantID,
		StoreId:           storeID,
		ProductId:         m.ProductID,
		VariantId:         variantID,
		Quantity:          m.Quantity,
		ReservedQuantity:  m.ReservedQuantity,
		AvailableQuantity: m.AvailableQuantity,
		ReorderPoint:      m.ReorderPoint,
		ReorderQuantity:   m.ReorderQuantity,
		LastCountedAt:     lastCounted,
		UpdatedAt:         timestamppb.New(m.UpdatedAt),
	}
}

func mapMovementToProto(m *model.InventoryMovement) *productv1.InventoryMovement {
	if m == nil {
		return nil
	}
	storeID := ""
	if m.StoreID != nil {
		storeID = *m.StoreID
	}
	variantID := ""
	if m.VariantID != nil {
		variantID = *m.VariantID
	}
	refType := ""
	if m.ReferenceType != nil {
		refType = *m.ReferenceType
	}
	refID := ""
	if m.ReferenceID != nil {
		refID = *m.ReferenceID
	}
	createdBy := ""
	if m.CreatedBy != nil {
		createdBy = *m.CreatedBy
	}

	return &productv1.InventoryMovement{
		Id:             m.ID,
		MerchantId:     m.MerchantID,
		StoreId:        storeID,
		ProductId:      m.ProductID,
		VariantId:      variantID,
		MovementType:   m.MovementType,
		QuantityChange: m.QuantityChange,
		QuantityBefore: m.QuantityBefore,
		QuantityAfter:  m.QuantityAfter,
		ReferenceType:  refType,
		ReferenceId:    refID,
		Notes:          m.Notes,
		CreatedBy:      createdBy,
		CreatedAt:      timestamppb.New(m.CreatedAt),
	}
}
