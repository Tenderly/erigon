package jsonrpc

import (
	"context"
	"github.com/tenderly/erigon/erigon-lib/common/hexutil"

	"github.com/tenderly/erigon/erigon-lib/common"

	"github.com/tenderly/erigon/eth/filters"

	"github.com/tenderly/erigon/erigon-lib/kv"

	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/p2p"
	"github.com/tenderly/erigon/rpc"
	"github.com/tenderly/erigon/turbo/rpchelper"
)

// ErigonAPI Erigon specific routines
type ErigonAPI interface {
	// System related (see ./erigon_system.go)
	Forks(ctx context.Context) (Forks, error)
	BlockNumber(ctx context.Context, rpcBlockNumPtr *rpc.BlockNumber) (hexutil.Uint64, error)

	// Blocks related (see ./erigon_blocks.go)
	GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	GetHeaderByHash(_ context.Context, hash common.Hash) (*types.Header, error)
	GetBlockByTimestamp(ctx context.Context, timeStamp rpc.Timestamp, fullTx bool) (map[string]interface{}, error)
	GetBalanceChangesInBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address]*hexutil.Big, error)

	// Receipt related (see ./erigon_receipts.go)
	GetLogsByHash(ctx context.Context, hash common.Hash) ([][]*types.Log, error)
	//GetLogsByNumber(ctx context.Context, number rpc.BlockNumber) ([][]*types.Log, error)
	GetLogs(ctx context.Context, crit filters.FilterCriteria) (types.ErigonLogs, error)
	GetLatestLogs(ctx context.Context, crit filters.FilterCriteria, logOptions filters.LogFilterOptions) (types.ErigonLogs, error)
	// Gets cannonical block receipt through hash. If the block is not cannonical returns error
	GetBlockReceiptsByBlockHash(ctx context.Context, cannonicalBlockHash common.Hash) ([]map[string]interface{}, error)

	// NodeInfo returns a collection of metadata known about the host.
	NodeInfo(ctx context.Context) ([]p2p.NodeInfo, error)
}

// ErigonImpl is implementation of the ErigonAPI interface
type ErigonImpl struct {
	*BaseAPI
	db         kv.RoDB
	ethBackend rpchelper.ApiBackend
}

// NewErigonAPI returns ErigonImpl instance
func NewErigonAPI(base *BaseAPI, db kv.RoDB, eth rpchelper.ApiBackend) *ErigonImpl {
	return &ErigonImpl{
		BaseAPI:    base,
		db:         db,
		ethBackend: eth,
	}
}
