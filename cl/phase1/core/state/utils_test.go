package state

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/cltypes"
	"github.com/tenderly/erigon/cl/cltypes/solid"
	"github.com/tenderly/erigon/cl/utils"
)

func TestValidatorSlashing(t *testing.T) {
	state := New(&clparams.MainnetBeaconConfig)
	utils.DecodeSSZSnappy(state, stateEncoded, int(clparams.DenebVersion))

	require.NoError(t, state.SlashValidator(1, nil))
	require.NoError(t, state.SlashValidator(2, nil))

	exit, err := state.BeaconState.ValidatorExitEpoch(1)
	require.NoError(t, err)
	require.NotEqual(t, exit, uint64(math.MaxUint64))
}

func TestValidatorFromDeposit(t *testing.T) {
	validator := ValidatorFromDeposit(&clparams.MainnetBeaconConfig, &cltypes.Deposit{
		Proof: solid.NewHashList(33),
		Data: &cltypes.DepositData{
			PubKey: [48]byte{69},
			Amount: 99999,
		},
	})
	require.Equal(t, validator.PublicKey(), [48]byte{69})
}

func TestSyncReward(t *testing.T) {
	state := New(&clparams.MainnetBeaconConfig)
	utils.DecodeSSZSnappy(state, stateEncoded, int(clparams.Phase0Version))

	propReward, partReward, err := state.SyncRewards()
	require.NoError(t, err)

	require.Equal(t, partReward, uint64(0x190))
	require.Equal(t, propReward, uint64(0x39))

}
