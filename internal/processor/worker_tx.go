package processor

import (
	"context"
	"encoding/json"
	"time"

	"eon/platform/internal/dedupe"
	"eon/platform/internal/event"
	"eon/platform/internal/log"
	"eon/platform/internal/outbox"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//handler inside postgres
type TxHandler interface {
	HandleTx(ctx context.Context,tx pgx.Tx,ev event.Event) error
}

//configure transact worker
type TxWorkerConfig struct {
	ConsumerID   string
	PartitionKey string

	DedupeWindow time.Duration

	Handler TxHandler

	DedupeStore dedupe.Store
	EventLog    log.EventLog
	Outbox      *outbox.Store
	OutboxTopic string

	DB *pgxpool.Pool

	CheckpointStore CheckpointStore
}

type TxWorker struct {
	cfg TxWorkerConfig
}

func NewTxWorker(cfg TxWorkerConfig) *TxWorker {
	if cfg.DedupeWindow == 0 {
		cfg.DedupeWindow=24 * time.Hour
	}
	return &TxWorker{cfg: cfg}
}

//begin consumption.
func (w *TxWorker) Run(ctx context.Context,fromPosition int64) error {
	start:=fromPosition
	if w.cfg.CheckpointStore!=nil {
		if pos,err:=w.cfg.CheckpointStore.Get(ctx,w.cfg.ConsumerID,w.cfg.PartitionKey); err == nil && pos > start {
			start=pos
		}
	}
	ch,stop,err:=w.cfg.EventLog.Subscribe(w.cfg.PartitionKey,start)
	if err!=nil {
		return err
	}
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case entry,ok:=<-ch:
			if !ok {
				return nil
			}
			if err:=w.processOneTx(ctx,entry); err!=nil {
				// In a real system: DLQ + backoff.
				continue
			}
			if w.cfg.CheckpointStore!=nil {
				_=w.cfg.CheckpointStore.Commit(ctx,w.cfg.ConsumerID,w.cfg.PartitionKey,entry.Position)
			}
		}
	}
}
func (w *TxWorker) processOneTx(ctx context.Context,entry log.LogEntry) error {
	var ev event.Event
	if err:=json.Unmarshal(entry.Payload,&ev); err!=nil {
		return err
	}
	res,err:=w.cfg.DedupeStore.Reserve(ctx,w.cfg.ConsumerID,ev.IdempotencyKey,ev.EventID,w.cfg.DedupeWindow)
	if err!=nil {
		return err
	}
	switch res {
	case dedupe.ResultDuplicate:
		return nil
	case dedupe.ResultReserved:
		return errReserved
	case dedupe.ResultNew:
	default:
		return errUnknownResult
	}

	tx,err:=w.cfg.DB.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil {
		return err
	}
	defer func() {
		_=tx.Rollback(ctx)
	}()


	if err:=w.cfg.Handler.HandleTx(ctx,tx,ev); err!=nil {
		return err
	}

	//we insert in same tx with outbox
	if w.cfg.Outbox!=nil && w.cfg.OutboxTopic!="" {
		if err:=w.cfg.Outbox.InsertTx(ctx,tx,ev.EventID,w.cfg.OutboxTopic,ev.Payload); err!=nil {
			return err
		}
	}

	//check processed keys dedupe
	table:="processed_keys"
	if ts,ok:=w.cfg.DedupeStore.(interface{ Table() string }); ok {
		table=ts.Table()
	}
	if err:=dedupe.MarkProcessedTx(ctx,tx,table,w.cfg.ConsumerID,ev.IdempotencyKey,ev.EventID); err!=nil {
		return err
	}

	if err:=tx.Commit(ctx); err!=nil {
		return err
	}

	return nil
}

