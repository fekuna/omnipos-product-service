package listener

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/fekuna/omnipos-product-service/internal/inventory"
	"github.com/fekuna/omnipos-product-service/internal/inventory/dto"
	"go.uber.org/zap"
)

type InventoryListener struct {
	consumer *broker.KafkaConsumer
	uc       inventory.UseCase
	logger   logger.ZapLogger
}

func NewInventoryListener(consumer *broker.KafkaConsumer, uc inventory.UseCase, logger logger.ZapLogger) *InventoryListener {
	return &InventoryListener{
		consumer: consumer,
		uc:       uc,
		logger:   logger,
	}
}

func (l *InventoryListener) Start(ctx context.Context) {
	l.logger.Info("Starting Inventory Kafka Listener")
	for {
		select {
		case <-ctx.Done():
			l.logger.Info("Stopping Inventory Kafka Listener")
			return
		default:
			msg, err := l.consumer.ReadMessage(ctx)
			if err != nil {
				// Don't log context canceled error as error
				if ctx.Err() != nil {
					return
				}
				l.logger.Error("Failed to read kafka message", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}
			l.processMessage(ctx, msg.Value)
		}
	}
}

type OrderCreatedEvent struct {
	EventID   string       `json:"event_id"`
	EventType string       `json:"event_type"`
	Payload   OrderPayload `json:"payload"`
	Timestamp time.Time    `json:"timestamp"`
}

type OrderPayload struct {
	ID         string             `json:"id"`
	MerchantID string             `json:"merchant_id"`
	StoreID    string             `json:"store_id"`
	Items      []OrderItemPayload `json:"items"`
}

type OrderItemPayload struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id"`
	Quantity  float64 `json:"quantity"`
}

func (l *InventoryListener) processMessage(ctx context.Context, value []byte) {
	var event OrderCreatedEvent
	if err := json.Unmarshal(value, &event); err != nil {
		l.logger.Error("Failed to unmarshal event", zap.Error(err))
		return
	}

	if event.EventType != "OrderCreated" {
		return
	}

	l.logger.Info("Processing OrderCreated event", zap.String("order_id", event.Payload.ID))

	for _, item := range event.Payload.Items {
		input := &dto.AdjustInventoryInput{
			MerchantID:     event.Payload.MerchantID,
			StoreID:        &event.Payload.StoreID,
			ProductID:      item.ProductID,
			VariantID:      item.VariantID,
			QuantityChange: -item.Quantity, // Deduction
			Reason:         "Order Sale",
			ReferenceID:    event.Payload.ID,
			ReferenceType:  "sale",
			UserID:         "system",
		}

		_, err := l.uc.AdjustInventory(ctx, input)
		if err != nil {
			l.logger.Error("Failed to adjust inventory for order item",
				zap.String("order_id", event.Payload.ID),
				zap.String("product_id", item.ProductID),
				zap.Error(err),
			)
			// TODO: Retry?
		}
	}
}
