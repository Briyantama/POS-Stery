package infrastructure

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/domain"
)

// NATSPublisher implements application.EventPublisher using NATS JetStream.
type NATSPublisher struct {
	js nats.JetStreamContext
}

// NewNATSPublisher creates a NATSPublisher backed by the provided JetStream context.
func NewNATSPublisher(js nats.JetStreamContext) *NATSPublisher {
	return &NATSPublisher{js: js}
}

// PublishSaleCompleted emits a Sale.Completed event to the POS_EVENTS stream.
// Failures are non-fatal from the caller's perspective (best-effort delivery).
func (p *NATSPublisher) PublishSaleCompleted(ctx context.Context, sale *domain.Sale) error {
	type salePayload struct {
		SaleID    string  `json:"sale_id"`
		CashierID string  `json:"cashier_id"`
		Total     float64 `json:"total"`
		Status    string  `json:"status"`
	}

	payload := salePayload{
		SaleID:    sale.ID.String(),
		CashierID: sale.CashierID.String(),
		Total:     sale.Total,
		Status:    sale.Status,
	}

	storeID := sale.StoreID.String()
	if err := sharednats.Publish(
		p.js,
		sharednats.SubjectSaleCompleted,
		sale.TenantID.String(),
		storeID,
		payload,
	); err != nil {
		return fmt.Errorf("publish Sale.Completed: %w", err)
	}
	return nil
}
