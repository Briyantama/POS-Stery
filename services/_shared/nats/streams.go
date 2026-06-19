package nats

import (
	"errors"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	StreamName = "POS_EVENTS"

	SubjectLowStock          = "Product.LowStock"
	SubjectSaleCompleted     = "Sale.Completed"
	SubjectInventoryReplenished = "Inventory.Replenished"
	SubjectCustomerNew       = "Customer.New"
	SubjectAIOrderForecasted = "AI.OrderForecasted"
)

// EnsureStreams creates the POS_EVENTS JetStream stream if it doesn't exist.
// Called at service startup — idempotent.
func EnsureStreams(js nats.JetStreamContext) error {
	_, err := js.StreamInfo(StreamName)
	if err == nil {
		return nil // already exists
	}

	if !errors.Is(err, nats.ErrStreamNotFound) {
		return err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name: StreamName,
		Subjects: []string{
			SubjectLowStock,
			SubjectSaleCompleted,
			SubjectInventoryReplenished,
			SubjectCustomerNew,
			SubjectAIOrderForecasted,
		},
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Replicas:  1,
	})
	return err
}
