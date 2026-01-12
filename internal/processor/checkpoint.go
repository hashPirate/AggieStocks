package processor

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


type CheckpointStore interface {
	Get(ctx context.Context,consumerID,partitionKey string) (int64,error)
	Commit(ctx context.Context,consumerID,partitionKey string,position int64) error
}

type PostgresCheckpointStore struct {
	pool  *pgxpool.Pool
	table string
}

func NewPostgresCheckpointStore(pool *pgxpool.Pool,table string) *PostgresCheckpointStore {
	return &PostgresCheckpointStore{pool: pool,table: table}
}

func (s *PostgresCheckpointStore) EnsureSchema(ctx context.Context) error {
	ddl:=`
CREATE TABLE IF NOT EXISTS ` + s.table + ` (
	consumer_id TEXT NOT NULL,
	partition_key TEXT NOT NULL,
	last_position BIGINT NOT NULL,
	PRIMARY KEY (consumer_id,partition_key)
);`
	_,err:=s.pool.Exec(ctx,ddl)
	return err
}

func (s *PostgresCheckpointStore) Get(ctx context.Context,consumerID,partitionKey string) (int64,error) {
	var pos int64
	err:=s.pool.QueryRow(ctx,
		`SELECT last_position FROM `+s.table+` WHERE consumer_id=$1 AND partition_key=$2`,
		consumerID,partitionKey,
	).Scan(&pos)
	if err!=nil {
		if errors.Is(err,pgx.ErrNoRows) {
			return 0,nil
		}
		return 0,err
	}
	return pos,nil
}

func (s *PostgresCheckpointStore) Commit(ctx context.Context,consumerID,partitionKey string,position int64) error {
	_,err:=s.pool.Exec(ctx,
		`INSERT INTO `+s.table+` (consumer_id,partition_key,last_position)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (consumer_id,partition_key) DO UPDATE
		 SET last_position=EXCLUDED.last_position`,
		consumerID,partitionKey,position,
	)
	return err
}

