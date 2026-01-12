package ledger

import (
	"context"
	"encoding/json"

	"eon/platform/internal/event"
	"github.com/jackc/pgx/v5"
)

//handler maintains a balance per account
//the types change the balancce
type Handler struct{}

func (h *Handler) HandleTx(ctx context.Context,tx pgx.Tx,ev event.Event) error {
	_,err:=tx.Exec(ctx,`
CREATE TABLE IF NOT EXISTS ledger (
	account_id TEXT PRIMARY KEY,
	balance BIGINT NOT NULL DEFAULT 0
);`)
	if err!=nil {
		return err
	}

	var payload struct {
		Amount int64 `json:"amount"`
	}
	if err:=json.Unmarshal(ev.Payload,&payload); err!=nil {
		return err
	}

	var delta int64
	switch ev.Type {
	case "charge.created":
		delta=payload.Amount
	case "refund.updated":
		delta=-payload.Amount
	default:
		return nil
	}

	_,err=tx.Exec(ctx,`
INSERT INTO ledger (account_id,balance)
VALUES ($1,$2)
ON CONFLICT (account_id) DO UPDATE
SET balance=ledger.balance + EXCLUDED.balance;`,
		ev.EntityKey,delta,
	)
	return err
}
