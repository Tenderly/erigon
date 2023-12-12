package bor

import (
	"github.com/idrecun/erigon/consensus"
	"github.com/idrecun/erigon/consensus/ethash"
	"github.com/idrecun/erigon/core/state"
	"github.com/idrecun/erigon/core/types"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/log/v3"
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
