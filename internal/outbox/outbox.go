package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxEntry struct {
	ID            int64
	EventID       string
	Topic         string
	Payload       json.RawMessage
	CreatedAt     time.Time
	PublishedAt   *time.Time
	LastError     *string
}
const createOutboxTable=`
CREATE TABLE IF NOT EXISTS outbox (
	id BIGSERIAL PRIMARY KEY,
	event_id TEXT NOT NULL,
	topic TEXT NOT NULL,
	payload JSONB NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	published_at TIMESTAMPTZ,
	last_error TEXT
);
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox(published_at) WHERE published_at IS NULL;
`
type Store struct {
	pool  *pgxpool.Pool
	table string
}

func NewStore(pool *pgxpool.Pool,table string) *Store {
	return &Store{pool: pool,table: table}
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	_,err:=s.pool.Exec(ctx,createOutboxTable)
	return err
}

func (s *Store) Insert(ctx context.Context,eventID,topic string,payload json.RawMessage) error {
	_,err:=s.pool.Exec(ctx,
		`INSERT INTO `+s.table+` (event_id,topic,payload,created_at) VALUES ($1,$2,$3,NOW())`,
		eventID,topic,payload,
	)
	return err
}

//add an outbox row using existing transaction
func (s *Store) InsertTx(ctx context.Context,tx pgx.Tx,eventID,topic string,payload json.RawMessage) error {
	_,err:=tx.Exec(ctx,
		`INSERT INTO `+s.table+` (event_id,topic,payload,created_at) VALUES ($1,$2,$3,NOW())`,
		eventID,topic,payload,
	)
	return err
}
func (s *Store) MarkPublished(ctx context.Context,eventID string) error {
	_,err:=s.pool.Exec(ctx,
		`UPDATE `+s.table+` SET published_at=NOW() WHERE event_id=$1 AND published_at IS NULL`,
		eventID,
	)
	return err
}

func (s *Store) MarkFailed(ctx context.Context,eventID string,errMsg string) error {
	_,err:=s.pool.Exec(ctx,
		`UPDATE `+s.table+` SET last_error=$2 WHERE event_id=$1`,
		eventID,errMsg,
	)
	return err
}
func (s *Store) Unpublished(ctx context.Context,limit int) ([]OutboxEntry,error) {
	rows,err:=s.pool.Query(ctx,
		`SELECT id,event_id,topic,payload,created_at,published_at,last_error FROM `+s.table+
			` WHERE published_at IS NULL ORDER BY id LIMIT $1`,limit,
	)
	if err!=nil {
		return nil,err
	}
	defer rows.Close()
	var out []OutboxEntry
	for rows.Next() {
		var e OutboxEntry
		var payload []byte
		err:=rows.Scan(&e.ID,&e.EventID,&e.Topic,&payload,&e.CreatedAt,&e.PublishedAt,&e.LastError)
		if err!=nil {
			return nil,err
		}
		e.Payload=payload
		out=append(out,e)
	}
	return out,rows.Err()
}
