package dedupe

import (
	"context"
	"testing"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReserveThenMarkProcessed - pretty self explanatory but provides a good test framework.
func TestReserveThenMarkProcessed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx:=context.Background()
	rdb:=redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()
	pool,err:=pgxpool.New(ctx,"postgres://postgres:postgres@localhost:5432/eon?sslmode=disable")
	require.NoError(t,err)
	defer pool.Close()

	store:=NewRedisPostgresStore(rdb,pool,"processed_keys")
	require.NoError(t,store.EnsureSchema(ctx))

	consumerID:="test-consumer"
	idemKey:="idem-" + time.Now().Format("20060102150405.000")
	eventID:="ev-1"
	window:=10 * time.Second

	// First time: New
	res,err:=store.Reserve(ctx,consumerID,idemKey,eventID,window)
	require.NoError(t,err)
	assert.Equal(t,ResultNew,res)

	// Mark processed
	err=store.MarkProcessed(ctx,consumerID,idemKey,eventID,window)
	require.NoError(t,err)

	// Second time: Duplicate
	res2,err:=store.Reserve(ctx,consumerID,idemKey,"ev-2",window)
	require.NoError(t,err)
	assert.Equal(t,ResultDuplicate,res2)

	seen,evID,err:=store.IsProcessed(ctx,consumerID,idemKey)
	require.NoError(t,err)
	assert.True(t,seen)
	assert.Equal(t,eventID,evID)
}
