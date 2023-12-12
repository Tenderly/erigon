package cache

import (
	"github.com/tenderly/erigon/cl/cltypes/solid"
	"github.com/tenderly/erigon/cl/phase1/core/state/lru"
	"github.com/tenderly/erigon/cl/utils"
	"github.com/tenderly/erigon/erigon-lib/common"
)

var attestationIndiciesCache *lru.Cache[common.Hash, []uint64]

const attestationIndiciesCacheSize = 1024

func LoadAttestatingIndicies(attestation *solid.AttestationData, aggregationBits []byte) ([]uint64, bool) {
	bitsHash := utils.Sha256(aggregationBits)
	hash, err := attestation.HashSSZ()
	if err != nil {
		return nil, false
	}
	return attestationIndiciesCache.Get(utils.Sha256(hash[:], bitsHash[:]))
}

func StoreAttestation(attestation *solid.AttestationData, aggregationBits []byte, indicies []uint64) {
	bitsHash := utils.Sha256(aggregationBits)
	hash, err := attestation.HashSSZ()
	if err != nil {
		return
	}
	attestationIndiciesCache.Add(utils.Sha256(hash[:], bitsHash[:]), indicies)
}
