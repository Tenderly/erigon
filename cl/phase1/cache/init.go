package cache

import (
	"github.com/tenderly/erigon/cl/phase1/core/state/lru"
	"github.com/tenderly/erigon/erigon-lib/common"
)

func init() {
	var err error
	if attestationIndiciesCache, err = lru.New[common.Hash, []uint64]("attestationIndiciesCacheSize", attestationIndiciesCacheSize); err != nil {
		panic(err)
	}
}
