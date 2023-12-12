package fromdb

import (
	"context"

	"github.com/tenderly/erigon/cmd/hack/tool"
	"github.com/tenderly/erigon/erigon-lib/chain"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/erigon-lib/kv/kvcfg"
	"github.com/tenderly/erigon/ethdb/prune"
)

func ChainConfig(db kv.RoDB) (cc *chain.Config) {
	err := db.View(context.Background(), func(tx kv.Tx) error {
		cc = tool.ChainConfig(tx)
		return nil
	})
	tool.Check(err)
	if cc == nil {
		panic("database is not initialized")
	}
	return cc
}

func PruneMode(db kv.RoDB) (pm prune.Mode) {
	if err := db.View(context.Background(), func(tx kv.Tx) error {
		var err error
		pm, err = prune.Get(tx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return
}
func HistV3(db kv.RoDB) (enabled bool) {
	if err := db.View(context.Background(), func(tx kv.Tx) error {
		var err error
		enabled, err = kvcfg.HistoryV3.Enabled(tx)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		panic(err)
	}
	return
}
