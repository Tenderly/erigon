package jsonrpc

import (
	"context"
	"errors"
	"github.com/tenderly/erigon/erigon-lib/common/hexutil"

	"github.com/tenderly/erigon/erigon-lib/common"

	borfinality "github.com/tenderly/erigon/consensus/bor/finality"
	"github.com/tenderly/erigon/consensus/bor/finality/whitelist"
	"github.com/tenderly/erigon/core/forkid"
	"github.com/tenderly/erigon/rpc"
	"github.com/tenderly/erigon/turbo/rpchelper"
)

// Forks is a data type to record a list of forks passed by this node
type Forks struct {
	GenesisHash common.Hash `json:"genesis"`
	HeightForks []uint64    `json:"heightForks"`
	TimeForks   []uint64    `json:"timeForks"`
}

// Forks implements erigon_forks. Returns the genesis block hash and a sorted list of all forks block numbers
func (api *ErigonImpl) Forks(ctx context.Context) (Forks, error) {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return Forks{}, err
	}
	defer tx.Rollback()

	chainConfig, genesis, err := api.chainConfigWithGenesis(tx)
	if err != nil {
		return Forks{}, err
	}
	heightForks, timeForks := forkid.GatherForks(chainConfig, genesis.Time())

	return Forks{genesis.Hash(), heightForks, timeForks}, nil
}

// Post the merge eth_blockNumber will return latest forkChoiceHead block number
// erigon_blockNumber will return latest executed block number or any block number requested
func (api *ErigonImpl) BlockNumber(ctx context.Context, rpcBlockNumPtr *rpc.BlockNumber) (hexutil.Uint64, error) {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var rpcBlockNum rpc.BlockNumber
	if rpcBlockNumPtr == nil {
		rpcBlockNum = rpc.LatestExecutedBlockNumber
	} else {
		rpcBlockNum = *rpcBlockNumPtr
	}

	var blockNum uint64
	switch rpcBlockNum {
	case rpc.LatestBlockNumber:
		blockNum, err = rpchelper.GetLatestBlockNumber(tx)
		if err != nil {
			return 0, err
		}
	case rpc.EarliestBlockNumber:
		blockNum = 0
	case rpc.SafeBlockNumber:
		blockNum, err = rpchelper.GetSafeBlockNumber(tx)
		if err != nil {
			return 0, err
		}
	case rpc.FinalizedBlockNumber:
		if whitelist.GetWhitelistingService() != nil {
			num := borfinality.GetFinalizedBlockNumber(tx)
			if num == 0 {
				return 0, errors.New("no finalized block")
			}

			blockNum = borfinality.CurrentFinalizedBlock(tx, num).NumberU64()
			return hexutil.Uint64(blockNum), nil
		}

		blockNum, err = rpchelper.GetFinalizedBlockNumber(tx)
		if err != nil {
			return 0, err
		}
	default:
		blockNum, err = rpchelper.GetLatestExecutedBlockNumber(tx)
		if err != nil {
			return 0, err
		}
	}

	return hexutil.Uint64(blockNum), nil
}
