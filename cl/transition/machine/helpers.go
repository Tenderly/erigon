package machine

import (
	"github.com/tenderly/erigon/cl/abstract"
	"github.com/tenderly/erigon/cl/cltypes"
	"github.com/tenderly/erigon/cl/phase1/core/state"
	"github.com/tenderly/erigon/erigon-lib/common"
)

func executionEnabled(s abstract.BeaconState, payload *cltypes.Eth1Block) bool {
	return (!state.IsMergeTransitionComplete(s) && payload.BlockHash != common.Hash{}) || state.IsMergeTransitionComplete(s)
}
