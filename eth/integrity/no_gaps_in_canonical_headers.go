package integrity

import (
	"context"
	"fmt"
	"time"

	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/core/rawdb"
	"github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/eth/stagedsync/stages"
	"github.com/tenderly/erigon/turbo/services"
	"github.com/tenderly/erigon/turbo/snapshotsync/freezeblocks"
)

func NoGapsInCanonicalHeaders(tx kv.Tx, ctx context.Context, br services.BlockReader) {
	logEvery := time.NewTicker(10 * time.Second)
	defer logEvery.Stop()

	if err := br.(*freezeblocks.BlockReader).Integrity(ctx); err != nil {
		panic(err)
	}

	firstBlockInDB := br.(*freezeblocks.BlockReader).FrozenBlocks() + 1
	lastBlockNum, err := stages.GetStageProgress(tx, stages.Headers)
	if err != nil {
		panic(err)
	}

	for i := firstBlockInDB; i < lastBlockNum; i++ {
		hash, err := rawdb.ReadCanonicalHash(tx, i)
		if err != nil {
			panic(err)
		}
		if hash == (common.Hash{}) {
			err = fmt.Errorf("canonical marker not found: %d\n", i)
			panic(err)
		}
		header := rawdb.ReadHeader(tx, hash, i)
		if header == nil {
			err = fmt.Errorf("header not found: %d\n", i)
			panic(err)
		}
		body, _, _ := rawdb.ReadBody(tx, hash, i)
		if body == nil {
			err = fmt.Errorf("header not found: %d\n", i)
			panic(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-logEvery.C:
			log.Info("[integrity] NoGapsInCanonicalHeaders", "progress", fmt.Sprintf("%dK/%dK", i/1000, lastBlockNum/1000))
		default:
		}
	}
}
