package cltypes_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tenderly/erigon/cl/cltypes"
)

func TestParticipationBits(t *testing.T) {
	bits := cltypes.JustificationBits{}
	bits.DecodeSSZ([]byte{2}, 0)
	require.Equal(t, bits, cltypes.JustificationBits{false, true, false, false})
	require.Equal(t, bits.Byte(), byte(2))
}
