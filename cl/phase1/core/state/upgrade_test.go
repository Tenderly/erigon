package state

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/utils"
)

//go:embed tests/phase0.ssz_snappy
var stateEncoded []byte

func TestUpgradeAndExpectedWithdrawals(t *testing.T) {
	s := New(&clparams.MainnetBeaconConfig)
	utils.DecodeSSZSnappy(s, stateEncoded, int(clparams.Phase0Version))
	require.NoError(t, s.UpgradeToAltair())
	require.NoError(t, s.UpgradeToBellatrix())
	require.NoError(t, s.UpgradeToCapella())
	require.NoError(t, s.UpgradeToDeneb())
	// now WITHDRAWAAALLLLSSSS
	w := ExpectedWithdrawals(s)
	assert.Empty(t, w)

}
