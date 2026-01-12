package log

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"


	"github.com/jackc/pgx/v5/pgxpool"
)

//appnd only postgres table used for the log w mutex
type PostgresLog struct {
	pool   *pgxpool.Pool
	table  string
	mu     sync.Mutex
	closed bool
}

//event log 
func NewPostgresLog(pool *pgxpool.Pool,table string) *PostgresLog {
	return &PostgresLog{pool: pool,table: table}
}

func (p *PostgresLog) EnsureSchema(ctx context.Context) error {
	//the schema: id BIGSERIAL,partition_key TEXT,payload JSONB,created_at TIMESTAMPTZ.
	ddl:=fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id BIGSERIAL PRIMARY KEY,
	partition_key TEXT NOT NULL,
	payload JSONB NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_%s_partition_id ON %s(partition_key,id);
`,p.table,p.table,p.table)
	_,err:=p.pool.Exec(ctx,ddl)
	return err
}

func (p *PostgresLog) Append(event []byte) (int64,error) {
	return p.AppendToPartition("default",event)
}

func (p *PostgresLog) AppendToPartition(partitionKey string,event []byte) (int64,error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0,ErrClosed
	}
	ctx:=context.Background()
	var id int64
	err:=p.pool.QueryRow(ctx,
		`INSERT INTO `+p.table+` (partition_key,payload,created_at) VALUES ($1,$2,NOW()) RETURNING id`,
		partitionKey,event,
	).Scan(&id)
	return id,err
}


func (p *PostgresLog) Subscribe(partitionKey string,fromPosition int64) (<-chan LogEntry,func() error,error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil,nil,ErrClosed
	}
	p.mu.Unlock()

	ch:=make(chan LogEntry,64)
	ctx,cancel:=context.WithCancel(context.Background())
	stop:=func() error { cancel(); return nil }

	go func() {
		defer close(ch)
		pos:=fromPosition
		idleSleep:=50 * time.Millisecond
		const maxIdleSleep=2 * time.Second

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			rows,err:=p.pool.Query(ctx,
				`SELECT id,payload FROM `+p.table+` WHERE partition_key=$1 AND id > $2 ORDER BY id LIMIT 100`,
				partitionKey,pos,
			)
			if err!=nil {
				if ctx.Err()!=nil {
					return
				}
				time.Sleep(idleSleep)
				if idleSleep < maxIdleSleep {
					idleSleep *= 2
					if idleSleep > maxIdleSleep {
						idleSleep=maxIdleSleep
					}
				}
				continue
			}

			var hadRows bool
			for rows.Next() {
				hadRows=true
				var id int64
				var payload []byte
				if err:=rows.Scan(&id,&payload); err!=nil {
					continue
				}
				var raw json.RawMessage
				if err:=json.Unmarshal(payload,&raw); err == nil {
					payload=raw
				}
				select {
				case ch <- LogEntry{Position: id,Payload: payload}:
					pos=id
				case <-ctx.Done():
					rows.Close()
					return
				}
			}
			rows.Close()

			if !hadRows {
				time.Sleep(idleSleep)
				if idleSleep < maxIdleSleep {
					idleSleep *= 2
					if idleSleep > maxIdleSleep {
						idleSleep=maxIdleSleep
					}
				}
				continue
			}
			idleSleep=50 * time.Millisecond
		}
	}()

	return ch,stop,nil
}

func (p *PostgresLog) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed=true
	p.pool.Close()
	return nil
}
