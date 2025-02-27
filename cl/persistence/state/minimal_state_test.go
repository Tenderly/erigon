package state_accessors

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/cl/clparams"
	"github.com/tenderly/erigon/cl/cltypes"
)

func TestMinimalState(t *testing.T) {
	m := &MinimalBeaconState{
		Version:                      clparams.CapellaVersion,
		Eth1Data:                     &cltypes.Eth1Data{},
		Fork:                         &cltypes.Fork{},
		Eth1DepositIndex:             0,
		JustificationBits:            &cltypes.JustificationBits{},
		NextWithdrawalIndex:          0,
		NextWithdrawalValidatorIndex: 0,
	}
	var b bytes.Buffer
	if err := m.WriteTo(&b); err != nil {
		t.Fatal(err)
	}
	m2 := &MinimalBeaconState{}
	if err := m2.ReadFrom(&b); err != nil {
		t.Fatal(err)
	}

	require.Equal(t, m, m2)
}
