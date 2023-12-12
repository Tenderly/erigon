package antiquary

import (
	"context"
	"fmt"
	"testing"

	_ "embed"

	"github.com/ledgerwatch/log/v3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/cl/antiquary/tests"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/cltypes"
	state_accessors "github.com/tenderly/erigon/cl/persistence/state"
	"github.com/tenderly/erigon/cl/phase1/core/state"
	"github.com/tenderly/erigon/erigon-lib/common/datadir"
	"github.com/tenderly/erigon/erigon-lib/kv/memdb"
)

func runTest(t *testing.T, blocks []*cltypes.SignedBeaconBlock, preState, postState *state.CachingBeaconState) {
	db := memdb.NewTestDB(t)
	reader := tests.LoadChain(blocks, db)

	ctx := context.Background()
	vt := state_accessors.NewStaticValidatorTable()
	f := afero.NewMemMapFs()
	a := NewAntiquary(ctx, preState, vt, &clparams.MainnetBeaconConfig, datadir.New("/tmp"), nil, db, nil, reader, nil, log.New(), true, f)
	require.NoError(t, a.IncrementBeaconState(ctx, blocks[len(blocks)-1].Block.Slot+33))
	// TODO: add more meaning here, like checking db values, will do so once i see some bugs
}

func TestStateAntiquaryCapella(t *testing.T) {
	t.Skip()
	blocks, preState, postState := tests.GetCapellaRandom()
	runTest(t, blocks, preState, postState)
}

func TestStateAntiquaryBellatrix(t *testing.T) {
	t.Skip()
	blocks, preState, postState := tests.GetBellatrixRandom()
	fmt.Println(len(blocks))
	runTest(t, blocks, preState, postState)
}

func TestStateAntiquaryPhase0(t *testing.T) {
	t.Skip()
	blocks, preState, postState := tests.GetPhase0Random()
	runTest(t, blocks, preState, postState)
}
