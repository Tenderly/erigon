package bor

import (
	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/consensus"
	"github.com/tenderly/erigon/consensus/ethash"
	"github.com/tenderly/erigon/core/state"
	"github.com/tenderly/erigon/core/types"
	"github.com/tenderly/erigon/erigon-lib/chain"
)

type FakeBor struct {
	*ethash.FakeEthash
}

// NewFaker creates a bor consensus engine with a FakeEthash
func NewFaker() *FakeBor {
	return &FakeBor{
		FakeEthash: ethash.NewFaker(),
	}
}

func (f *FakeBor) Finalize(config *chain.Config, header *types.Header, state *state.IntraBlockState,
	txs types.Transactions, uncles []*types.Header, r types.Receipts, withdrawals []*types.Withdrawal,
	chain consensus.ChainReader, syscall consensus.SystemCall, logger log.Logger,
) (types.Transactions, types.Receipts, error) {
	return f.FakeEthash.Finalize(config, header, state, txs, uncles, r, withdrawals, chain, syscall, logger)
}
