package dedupe

import (
	"context"
	"time"
)

type Result int

const (
	ResultUnknown Result=iota
	ResultNew     
	ResultDuplicate 
	ResultReserved  
)

//record of a processed idempotency key   
type Record struct {
	ConsumerID     string
	IdempotencyKey string
	EventID        string
	ProcessedAt    time.Time
	//time to store the key.
	DedupeWindow time.Duration
}

type Store interface {
	Reserve(ctx context.Context,consumerID,idempotencyKey,eventID string,window time.Duration) (Result,error)
	MarkProcessed(ctx context.Context,consumerID,idempotencyKey,eventID string,window time.Duration) error
	IsProcessed(ctx context.Context,consumerID,idempotencyKey string) (bool,string,error)
}
