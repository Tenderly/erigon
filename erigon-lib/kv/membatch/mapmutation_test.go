package membatch

import (
	"context"
	"os"
	"testing"

	"github.com/ledgerwatch/log/v3"
	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/erigon-lib/kv/memdb"
)

func TestMapmutation_Flush_Close(t *testing.T) {
	db := memdb.NewTestDB(t)

	tx, err := db.BeginRw(context.Background())
	require.NoError(t, err)
	defer tx.Rollback()

	batch := NewHashBatch(tx, nil, os.TempDir(), log.New())
	defer func() {
		batch.Close()
	}()
	err = batch.Put(kv.ChaindataTables[0], []byte{1}, []byte{1})
	require.NoError(t, err)
	err = batch.Put(kv.ChaindataTables[0], []byte{2}, []byte{2})
	require.NoError(t, err)
	err = batch.Flush(context.Background(), tx)
	require.NoError(t, err)
	batch.Close()
	batch.Close()
}
