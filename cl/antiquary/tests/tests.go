package tests

import (
	"context"
	"embed"
	_ "embed"
	"strconv"

	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/cltypes"
	"github.com/tenderly/erigon/cl/persistence/beacon_indicies"
	"github.com/tenderly/erigon/cl/phase1/core/state"
	"github.com/tenderly/erigon/cl/utils"
	libcommon "github.com/tenderly/erigon/erigon-lib/common"
	"github.com/tenderly/erigon/erigon-lib/kv"
)

//go:embed test_data/capella/blocks_0.ssz_snappy
var capella_blocks_0_ssz_snappy []byte

//go:embed test_data/capella/blocks_1.ssz_snappy
var capella_blocks_1_ssz_snappy []byte

//go:embed test_data/capella/pre.ssz_snappy
var capella_pre_state_ssz_snappy []byte

//go:embed test_data/capella/post.ssz_snappy
var capella_post_state_ssz_snappy []byte

//go:embed test_data/phase0/blocks_0.ssz_snappy
var phase0_blocks_0_ssz_snappy []byte

//go:embed test_data/phase0/blocks_1.ssz_snappy
var phase0_blocks_1_ssz_snappy []byte

//go:embed test_data/phase0/pre.ssz_snappy
var phase0_pre_state_ssz_snappy []byte

//go:embed test_data/phase0/post.ssz_snappy
var phase0_post_state_ssz_snappy []byte

// bellatrix is long

//go:embed test_data/bellatrix
var bellatrixFS embed.FS

type MockBlockReader struct {
	u map[uint64]*cltypes.SignedBeaconBlock
}

func NewMockBlockReader() *MockBlockReader {
	return &MockBlockReader{u: make(map[uint64]*cltypes.SignedBeaconBlock)}
}

func (m *MockBlockReader) ReadBlockBySlot(ctx context.Context, tx kv.Tx, slot uint64) (*cltypes.SignedBeaconBlock, error) {
	return m.u[slot], nil
}

func (m *MockBlockReader) ReadBlockByRoot(ctx context.Context, tx kv.Tx, blockRoot libcommon.Hash) (*cltypes.SignedBeaconBlock, error) {
	panic("implement me")
}
func (m *MockBlockReader) ReadHeaderByRoot(ctx context.Context, tx kv.Tx, blockRoot libcommon.Hash) (*cltypes.SignedBeaconBlockHeader, error) {
	panic("implement me")
}

func (m *MockBlockReader) FrozenSlots() uint64 {
	panic("implement me")
}

func LoadChain(blocks []*cltypes.SignedBeaconBlock, db kv.RwDB) *MockBlockReader {
	tx, err := db.BeginRw(context.Background())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()

	m := NewMockBlockReader()
	for _, block := range blocks {
		m.u[block.Block.Slot] = block
		h := block.SignedBeaconBlockHeader()
		if err := beacon_indicies.WriteBeaconBlockHeaderAndIndicies(context.Background(), tx, h, true); err != nil {
			panic(err)
		}
		if err := beacon_indicies.WriteHighestFinalized(tx, block.Block.Slot+64); err != nil {
			panic(err)
		}
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
	return m
}

func GetCapellaRandom() ([]*cltypes.SignedBeaconBlock, *state.CachingBeaconState, *state.CachingBeaconState) {
	block1 := cltypes.NewSignedBeaconBlock(&clparams.MainnetBeaconConfig)
	block2 := cltypes.NewSignedBeaconBlock(&clparams.MainnetBeaconConfig)

	// Lets do te
	if err := utils.DecodeSSZSnappy(block1, capella_blocks_0_ssz_snappy, int(clparams.CapellaVersion)); err != nil {
		panic(err)
	}
	if err := utils.DecodeSSZSnappy(block2, capella_blocks_1_ssz_snappy, int(clparams.CapellaVersion)); err != nil {
		panic(err)
	}

	preState := state.New(&clparams.MainnetBeaconConfig)
	if err := utils.DecodeSSZSnappy(preState, capella_pre_state_ssz_snappy, int(clparams.CapellaVersion)); err != nil {
		panic(err)

	}
	postState := state.New(&clparams.MainnetBeaconConfig)
	if err := utils.DecodeSSZSnappy(postState, capella_post_state_ssz_snappy, int(clparams.CapellaVersion)); err != nil {
		panic(err)
	}
	return []*cltypes.SignedBeaconBlock{block1, block2}, preState, postState
}

func GetPhase0Random() ([]*cltypes.SignedBeaconBlock, *state.CachingBeaconState, *state.CachingBeaconState) {
	block1 := cltypes.NewSignedBeaconBlock(&clparams.MainnetBeaconConfig)
	block2 := cltypes.NewSignedBeaconBlock(&clparams.MainnetBeaconConfig)

	// Lets do te
	if err := utils.DecodeSSZSnappy(block1, phase0_blocks_0_ssz_snappy, int(clparams.Phase0Version)); err != nil {
		panic(err)
	}
	if err := utils.DecodeSSZSnappy(block2, phase0_blocks_1_ssz_snappy, int(clparams.Phase0Version)); err != nil {
		panic(err)
	}

	preState := state.New(&clparams.MainnetBeaconConfig)
	if err := utils.DecodeSSZSnappy(preState, phase0_pre_state_ssz_snappy, int(clparams.Phase0Version)); err != nil {
		panic(err)
	}
	postState := state.New(&clparams.MainnetBeaconConfig)
	if err := utils.DecodeSSZSnappy(postState, phase0_post_state_ssz_snappy, int(clparams.Phase0Version)); err != nil {
		panic(err)
	}
	return []*cltypes.SignedBeaconBlock{block1, block2}, preState, postState
}

func GetBellatrixRandom() ([]*cltypes.SignedBeaconBlock, *state.CachingBeaconState, *state.CachingBeaconState) {
	ret := make([]*cltypes.SignedBeaconBlock, 0, 96)
	// format for blocks is blocks_{i}.ssz_snappy where i is the index of the block, starting from 0 to 95 included.
	for i := 0; i < 96; i++ {
		block := cltypes.NewSignedBeaconBlock(&clparams.MainnetBeaconConfig)
		// Lets do te
		b, err := bellatrixFS.ReadFile("test_data/bellatrix/blocks_" + strconv.FormatInt(int64(i), 10) + ".ssz_snappy")
		if err != nil {
			panic(err)
		}
		if err := utils.DecodeSSZSnappy(block, b, int(clparams.BellatrixVersion)); err != nil {
			panic(err)
		}
		ret = append(ret, block)
	}
	preState := state.New(&clparams.MainnetBeaconConfig)
	b, err := bellatrixFS.ReadFile("test_data/bellatrix/pre.ssz_snappy")
	if err != nil {
		panic(err)
	}
	if err := utils.DecodeSSZSnappy(preState, b, int(clparams.BellatrixVersion)); err != nil {
		panic(err)
	}
	postState := state.New(&clparams.MainnetBeaconConfig)
	b, err = bellatrixFS.ReadFile("test_data/bellatrix/post.ssz_snappy")
	if err != nil {
		panic(err)
	}
	if err := utils.DecodeSSZSnappy(postState, b, int(clparams.BellatrixVersion)); err != nil {
		panic(err)
	}
	return ret, preState, postState

}
