package jsonrpc

import (
	"context"
	"fmt"
	"github.com/tenderly/erigon/erigon-lib/common/hexutil"
	"math/big"

	"github.com/tenderly/erigon/core/rawdb"
	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/kv"
	"github.com/tenderly/erigon/rpc"
	"github.com/tenderly/erigon/turbo/adapter/ethapi"
	"github.com/tenderly/erigon/turbo/rpchelper"
)

type GraphQLAPI interface {
	GetBlockDetails(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error)
	GetChainID(ctx context.Context) (*big.Int, error)
}

type GraphQLAPIImpl struct {
	*BaseAPI
	db kv.RoDB
}

func NewGraphQLAPI(base *BaseAPI, db kv.RoDB) *GraphQLAPIImpl {
	return &GraphQLAPIImpl{
		BaseAPI: base,
		db:      db,
	}
}

func (api *GraphQLAPIImpl) GetChainID(ctx context.Context) (*big.Int, error) {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	response, err := api.chainConfig(tx)
	if err != nil {
		return nil, err
	}

	return response.ChainID, nil
}

func (api *GraphQLAPIImpl) GetBlockDetails(ctx context.Context, blockNumber rpc.BlockNumber) (map[string]interface{}, error) {
	tx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	block, senders, err := api.getBlockWithSenders(ctx, blockNumber, tx)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	getBlockRes, err := api.delegateGetBlockByNumber(tx, block, blockNumber, false)
	if err != nil {
		return nil, err
	}

	chainConfig, err := api.chainConfig(tx)
	if err != nil {
		return nil, err
	}

	receipts, err := api.getReceipts(ctx, tx, chainConfig, block, senders)
	if err != nil {
		return nil, fmt.Errorf("getReceipts error: %w", err)
	}

	result := make([]map[string]interface{}, 0, len(receipts))
	for _, receipt := range receipts {
		txn := block.Transactions()[receipt.TransactionIndex]

		transaction := marshalReceipt(receipt, txn, chainConfig, block.HeaderNoCopy(), txn.Hash(), true)
		transaction["nonce"] = txn.GetNonce()
		transaction["value"] = txn.GetValue()
		transaction["data"] = txn.GetData()
		transaction["logs"] = receipt.Logs
		result = append(result, transaction)
	}

	response := map[string]interface{}{}
	response["block"] = getBlockRes
	response["receipts"] = result

	return response, nil
}

func (api *GraphQLAPIImpl) getBlockWithSenders(ctx context.Context, number rpc.BlockNumber, tx kv.Tx) (*types.Block, []common.Address, error) {
	if number == rpc.PendingBlockNumber {
		return api.pendingBlock(), nil, nil
	}

	blockHeight, blockHash, _, err := rpchelper.GetBlockNumber(rpc.BlockNumberOrHashWithNumber(number), tx, api.filters)
	if err != nil {
		return nil, nil, err
	}

	block, senders, err := api._blockReader.BlockWithSenders(ctx, tx, blockHash, blockHeight)
	return block, senders, err
}

func (api *GraphQLAPIImpl) delegateGetBlockByNumber(tx kv.Tx, b *types.Block, number rpc.BlockNumber, inclTx bool) (map[string]interface{}, error) {
	td, err := rawdb.ReadTd(tx, b.Hash(), b.NumberU64())
	if err != nil {
		return nil, err
	}
	additionalFields := make(map[string]interface{})
	response, err := ethapi.RPCMarshalBlock(b, inclTx, inclTx, additionalFields)
	if !inclTx {
		delete(response, "transactions") // workaround for https://github.com/tenderly/erigon/issues/4989#issuecomment-1218415666
	}
	response["totalDifficulty"] = (*hexutil.Big)(td)
	response["transactionCount"] = b.Transactions().Len()

	if err == nil && number == rpc.PendingBlockNumber {
		// Pending blocks need to nil out a few fields
		for _, field := range []string{"hash", "nonce", "miner"} {
			response[field] = nil
		}
	}

	return response, err
}
