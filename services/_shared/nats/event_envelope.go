package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// EventEnvelope wraps all NATS events with tenant context and metadata.
// All services must use this type for publishing; never publish raw payloads.
type EventEnvelope struct {
	EventType  string    `json:"event_type"`
	TenantID   string    `json:"tenant_id"`
	StoreID    string    `json:"store_id"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    any       `json:"payload"`
}

// Publish serialises the envelope and publishes it to the given NATS subject.
func Publish(js nats.JetStreamContext, subject, tenantID, storeID string, payload any) error {
	env := EventEnvelope{
		EventType:  subject,
		TenantID:   tenantID,
		StoreID:    storeID,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal event %s: %w", subject, err)
	}

	if _, err := js.Publish(subject, data); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}

	return nil
}
