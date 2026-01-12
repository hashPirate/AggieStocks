package dedupe

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

//MarkProcessedTx can write the durable processed record using an existing transaction.
func MarkProcessedTx(ctx context.Context,tx pgx.Tx,table,consumerID,idempotencyKey,eventID string) error {
	_,err:=tx.Exec(ctx,
		`INSERT INTO `+table+` (consumer_id,idempotency_key,event_id,processed_at) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (consumer_id,idempotency_key) DO NOTHING`,
		consumerID,idempotencyKey,eventID,time.Now().UTC(),
	)
	return err
}

