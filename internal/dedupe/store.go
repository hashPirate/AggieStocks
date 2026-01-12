package dedupe

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	redisKeyPrefix  ="eon:dedupe:"
	redisReserveTTL =30 * time.Second
)

//Redis can be used for the fast reserve and postgres can be used. Append only non editable table.
type RedisPostgresStore struct {
	redis *redis.Client
	pg    *pgxpool.Pool
	table string
}

//double layer dedupe 
func NewRedisPostgresStore(rdb *redis.Client,pool *pgxpool.Pool,table string) *RedisPostgresStore {
	return &RedisPostgresStore{redis: rdb,pg: pool,table: table}
}
const createDedupeTable=`
CREATE TABLE IF NOT EXISTS processed_keys (
	consumer_id TEXT NOT NULL,
	idempotency_key TEXT NOT NULL,
	event_id TEXT NOT NULL,
	processed_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (consumer_id,idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_processed_keys_processed_at ON processed_keys(processed_at);
`

func (s *RedisPostgresStore) EnsureSchema(ctx context.Context) error { //verification
	_,err:=s.pg.Exec(ctx,createDedupeTable)
	return err
}

func redisKey(consumerID,idempotencyKey string) string {
	return redisKeyPrefix + consumerID + ":" + idempotencyKey
}

// set reserve to NX (this function was completely ai generated lol)
func (s *RedisPostgresStore) Reserve(ctx context.Context,consumerID,idempotencyKey,eventID string,window time.Duration) (Result,error) {
	key:=redisKey(consumerID,idempotencyKey)

	// Durable check first: if already in Postgres,it's a duplicate.
	seen,evID,err:=s.IsProcessed(ctx,consumerID,idempotencyKey)
	if err!=nil {
		return ResultUnknown,err
	}
	if seen {
		_=evID
		return ResultDuplicate,nil
	}

	// Fast-path: reserve in Redis (NX=only if not exists).
	ok,err:=s.redis.SetNX(ctx,key,eventID,redisReserveTTL).Result()
	if err!=nil {
		return ResultUnknown,err
	}
	if !ok {
		// Key exists: either we're racing (reserved) or already processed. Re-check Postgres.
		seen,_,err=s.IsProcessed(ctx,consumerID,idempotencyKey)
		if err!=nil {
			return ResultUnknown,err
		}
		if seen {
			return ResultDuplicate,nil
		}
		return ResultReserved,nil
	}
	return ResultNew,nil
}

// MarkProcessed!!
func (s *RedisPostgresStore) MarkProcessed(ctx context.Context,consumerID,idempotencyKey,eventID string,window time.Duration) error {
	_,err:=s.pg.Exec(ctx,
		`INSERT INTO `+s.table+` (consumer_id,idempotency_key,event_id,processed_at) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (consumer_id,idempotency_key) DO NOTHING`,
		consumerID,idempotencyKey,eventID,time.Now().UTC(),
	)
	if err!=nil {
		return err
	}
	key:=redisKey(consumerID,idempotencyKey)
	//faster duplicate detection if we keep redis in the reserve (good idea satvik)
	if window > redisReserveTTL {
		s.redis.Set(ctx,key,eventID,window)
	} else {
		s.redis.Set(ctx,key,eventID,redisReserveTTL)
	}
	return nil
}

// THIS IS OUR SOURCE OF TRUTH!! VERIFY VIA THIS FIRST
func (s *RedisPostgresStore) IsProcessed(ctx context.Context,consumerID,idempotencyKey string) (bool,string,error) {
	var eventID string
	err:=s.pg.QueryRow(ctx,
		`SELECT event_id FROM `+s.table+` WHERE consumer_id=$1 AND idempotency_key=$2`,
		consumerID,idempotencyKey,
	).Scan(&eventID)
	if err!=nil {
		if errors.Is(err,pgx.ErrNoRows) {
			return false,"",nil
		}
		return false,"",err
	}
	return true,eventID,nil
}

// 
func (s *RedisPostgresStore) Table() string { return s.table }

var _ Store=(*RedisPostgresStore)(nil)
