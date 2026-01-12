package processor

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
	"eon/platform/internal/dedupe"
	"eon/platform/internal/event"
	"eon/platform/internal/log"
	"eon/platform/internal/outbox"
)

type WorkerConfig struct {
	ConsumerID     string
	PartitionKey   string
	DedupeWindow   time.Duration
	Handler        Handler
	DedupeStore    dedupe.Store
	EventLog       log.EventLog
	Outbox         *outbox.Store
	OutboxTopic    string
	MaxRetries     int
	RetryBackoff   time.Duration
	CheckpointStore CheckpointStore
}

type Worker struct {
	cfg WorkerConfig
}

func NewWorker(cfg WorkerConfig) *Worker {
	if cfg.DedupeWindow == 0 {
		cfg.DedupeWindow=24 * time.Hour
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries=3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff=time.Second
	}
	return &Worker{cfg: cfg}
}

func (w *Worker) Run(ctx context.Context,fromPosition int64) error {
	ch,stop,err:=w.cfg.EventLog.Subscribe(w.cfg.PartitionKey,fromPosition)
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
			if err:=w.processOne(ctx,entry); err!=nil {
				slog.Error("process one failed","position",entry.Position,"error",err)
				continue
			}
			if w.cfg.CheckpointStore!=nil {
				if err:=w.cfg.CheckpointStore.Commit(ctx,w.cfg.ConsumerID,w.cfg.PartitionKey,entry.Position); err!=nil {
					slog.Error("failed to commit checkpoint","position",entry.Position,"error",err)
				}
			}
		}
	}
}

func (w *Worker) processOne(ctx context.Context,entry log.LogEntry) error {
	var ev event.Event
	if err:=json.Unmarshal(entry.Payload,&ev); err!=nil {
		return err
	}

	//dedupecheck:we should reserve before processing and marking processed.
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

	var lastErr error
	for attempt:=0; attempt < w.cfg.MaxRetries; attempt++ {
		lastErr=w.cfg.Handler.Handle(ctx,ev)
		if lastErr==nil {
			break
		}
		if attempt<w.cfg.MaxRetries-1 {
			time.Sleep(w.cfg.RetryBackoff * time.Duration(attempt+1))
		}
	}
	if lastErr!=nil {
		return lastErr
	}

	if w.cfg.Outbox!=nil && w.cfg.OutboxTopic!="" {
		_=w.cfg.Outbox.Insert(ctx,ev.EventID,w.cfg.OutboxTopic,ev.Payload)
	}

	return w.cfg.DedupeStore.MarkProcessed(ctx,w.cfg.ConsumerID,ev.IdempotencyKey,ev.EventID,w.cfg.DedupeWindow)
}

var (
	errReserved=errTyped{"key reserved by another"}
	errUnknownResult=errTyped{"unknown dedupe result"}
)

type errTyped struct{ msg string }

func (e errTyped) Error() string { return e.msg }
