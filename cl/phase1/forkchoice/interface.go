package forkchoice

import (
	"github.com/tenderly/erigon/cl/cltypes"
	"github.com/tenderly/erigon/cl/cltypes/solid"
	"github.com/tenderly/erigon/cl/phase1/core/state"
	"github.com/tenderly/erigon/cl/phase1/execution_client"
	"github.com/tenderly/erigon/erigon-lib/common"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
)

type ForkChoiceStorage interface {
	ForkChoiceStorageWriter
	ForkChoiceStorageReader
}

type ForkChoiceStorageReader interface {
	Ancestor(root common.Hash, slot uint64) common.Hash
	AnchorSlot() uint64
	Engine() execution_client.ExecutionEngine
	FinalizedCheckpoint() solid.Checkpoint
	FinalizedSlot() uint64
	GetEth1Hash(eth2Root common.Hash) common.Hash
	GetHead() (common.Hash, uint64, error)
	HighestSeen() uint64
	JustifiedCheckpoint() solid.Checkpoint
	JustifiedSlot() uint64
	ProposerBoostRoot() common.Hash
	GetStateAtBlockRoot(blockRoot libcommon.Hash, alwaysCopy bool) (*state.CachingBeaconState, error)
	Slot() uint64
	Time() uint64

	GetStateAtSlot(slot uint64, alwaysCopy bool) (*state.CachingBeaconState, error)
	GetStateAtStateRoot(root libcommon.Hash, alwaysCopy bool) (*state.CachingBeaconState, error)
}

type ForkChoiceStorageWriter interface {
	OnAttestation(attestation *solid.Attestation, fromBlock bool) error
	OnAttesterSlashing(attesterSlashing *cltypes.AttesterSlashing, test bool) error
	OnBlock(block *cltypes.SignedBeaconBlock, newPayload bool, fullValidation bool) error
	OnTick(time uint64)
}
