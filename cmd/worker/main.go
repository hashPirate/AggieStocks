package main

import (
	"context"
	"log"
	"os"
	"time"
	"eon/platform/internal/dedupe"
	elog "eon/platform/internal/log"
	"eon/platform/internal/outbox"
	"eon/platform/internal/processor"
	"eon/platform/internal/ledger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ledgerConfig struct {
	PgURL   string
	RedisURL string
}
func loadLedgerConfig() ledgerConfig {
	pgURL:=os.Getenv("DATABASE_URL")
	if pgURL == "" {
		pgURL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable"
	}
	redisURL:=os.Getenv("REDIS_URL")
	if redisURL=="" {redisURL="redis://localhost:6379"}
return ledgerConfig{PgURL: pgURL,RedisURL: redisURL}
}


func main() {
	cfg :=loadLedgerConfig()
	ctx:=context.Background()

	pool,err:=pgxpool.New(ctx,cfg.PgURL)
	if err!=nil {
		log.Fatal("pg pool:",err)
	}
	defer pool.Close()

	rdbOpt,err:=redis.ParseURL(cfg.RedisURL)
	if err!=nil {
		log.Fatal("redis parse:",err)
	}
	rdb:=redis.NewClient(rdbOpt)
	defer rdb.Close()

	// Event logging, dedupe store,the outbox,checkpoint store.
	eventLog:=elog.NewPostgresLog(pool,"event_log")
	if err:=eventLog.EnsureSchema(ctx); err!=nil {
		log.Fatal("event_log schema:",err)
	}

	dedupeStore:=dedupe.NewRedisPostgresStore(rdb,pool,"processed_keys")
	if err:= dedupeStore.EnsureSchema(ctx); err!=nil {
		log.Fatal("dedupe schema:",err)
	}

	outboxStore:=outbox.NewStore(pool,"outbox")
	if err:=outboxStore.EnsureSchema(ctx); err!=nil {
		log.Fatal("outbox schema:",err)
	}

	checkpoints:=processor.NewPostgresCheckpointStore(pool,"consumer_checkpoints")
	if err:=checkpoints.EnsureSchema(ctx); err!=nil {
		log.Fatal("checkpoint schema:",err)
	}

	worker:=processor.NewTxWorker(processor.TxWorkerConfig{
		ConsumerID:      "ledger-consumer",
		PartitionKey:    "acct_123",// example partitin im using
		DedupeWindow:    24 * time.Hour,
		Handler:        &ledger.Handler{},

		DedupeStore:     dedupeStore,
		EventLog:        eventLog,
		Outbox:          outboxStore,
		OutboxTopic:     "ledger.events",
		DB:              pool,
		CheckpointStore: checkpoints,
	})

	if err:=worker.Run(ctx,0); err!=nil {
		log.Fatal("worker run:",err)
	}
}

