package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {
	payload:=json.RawMessage(`{"amount":1000}`)
	ev:=NewEvent("idem-1","acc_123","charge.created",1,payload)
	assert.NotEmpty(t,ev.EventID)
	assert.Equal(t,"idem-1",ev.IdempotencyKey)
	assert.Equal(t,"acc_123",ev.EntityKey)
	assert.Equal(t,"charge.created",ev.Type)
	assert.Equal(t,1,ev.Version)
	assert.Equal(t,payload,ev.Payload)
	assert.False(t,ev.CreatedAt.IsZero())
}
