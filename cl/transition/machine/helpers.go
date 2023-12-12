package machine

import (
	"github.com/idrecun/erigon/cl/abstract"
	"github.com/idrecun/erigon/cl/cltypes"
	"github.com/idrecun/erigon/cl/phase1/core/state"
	"github.com/idrecun/erigon/erigon-lib/common"
)

func executionEnabled(s abstract.BeaconState, payload *cltypes.Eth1Block) bool {
	return (!state.IsMergeTransitionComplete(s) && payload.BlockHash != common.Hash{}) || state.IsMergeTransitionComplete(s)
}
