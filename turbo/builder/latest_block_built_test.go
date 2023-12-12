package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tenderly/erigon/core/types"
)

func TestLatestBlockBuilt(t *testing.T) {
	t.Parallel()
	s := NewLatestBlockBuiltStore()
	b := types.NewBlockWithHeader(&types.Header{})
	s.AddBlockBuilt(b)
	assert.Equal(t, b.Header(), s.BlockBuilt().Header())
}
