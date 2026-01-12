package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)


type Event struct {
	EventID       string          `json:"event_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	EntityKey     string          `json:"entity_key"`   //ordering key for account_id,payment_intent_id
	Type          string          `json:"type"`         //charge.created,payout.paid
	Version       int             `json:"version"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
}

//event made by id and timestamp
func NewEvent(idempotencyKey,entityKey,eventType string,version int,payload json.RawMessage) Event {
	return Event{
		EventID:        uuid.New().String(),
		IdempotencyKey: idempotencyKey,
		EntityKey:      entityKey,
		Type:           eventType,
		Version:        version,
		Payload:        payload,
		CreatedAt:      time.Now().UTC(),
	}
}

//for the ingress api
type IngressRequest struct {
	IdempotencyKey string          `json:"idempotency_key"`
	EntityKey      string          `json:"entity_key"`
	Type           string          `json:"type"`
	Version        int             `json:"version,omitempty"`
	Payload        json.RawMessage `json:"payload"`
}
