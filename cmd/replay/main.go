package main

import (
	"context"
	"flag"
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

func main() {
	var (
		from      =flag.Int64("from",0,"starting position to replay from")
		partition =flag.String("partition","","partition key to replay (ex account id)")
		consumerID= flag.String("consumer","ledger-consumer","consumer ID to use")
	)
	flag.Parse()
	if *partition == "" {
		log.Fatal("partition is required")
	}
	ctx:=context.Background()
	pgURL:=os.Getenv("DATABASE_URL")
	if pgURL == "" {
		pgURL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable"
	}
	redisURL:=os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL="redis://localhost:6379"
	}
	pool,err:=pgxpool.New(ctx,pgURL)
	if err!=nil {
		log.Fatal("pg pool:",err)
	}
	defer pool.Close()
	rdbOpt,err :=redis.ParseURL(redisURL)
	if err!=nil {
		log.Fatal("redis parse:",err)
	}
	rdb:=redis.NewClient(rdbOpt)
	defer rdb.Close()
	// keep on the same pipeline as worker but allow custom ones too
	eventLog:=elog.NewPostgresLog(pool,"event_log")
	if err:=eventLog.EnsureSchema(ctx);err!=nil {
		log.Fatal("event_log schema:",err)
	}
	dedupeStore:=dedupe.NewRedisPostgresStore(rdb,pool,"processed_keys")
	if err:=dedupeStore.EnsureSchema(ctx); err!=nil {
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

	worker:=processor.NewTxWorker(processor.TxWorkerConfig{ //added the new ledger system to make sure we have the compile fix
		ConsumerID:      *consumerID,
		PartitionKey:    *partition,
		DedupeWindow:    24 * time.Hour,
		Handler:         &ledger.Handler{},
		DedupeStore:     dedupeStore,
		EventLog:        eventLog,
		Outbox:          outboxStore,
		OutboxTopic:     "ledger.events",
		DB:              pool,
		CheckpointStore: checkpoints,
	})

	if err:=worker.Run(ctx,*from); err!=nil {
		log.Fatal("replay run:",err)
	}
}

