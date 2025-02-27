package tool

import (
	"strconv"

	"github.com/tenderly/erigon/core/rawdb"
	"github.com/tenderly/erigon/erigon-lib/chain"
	"github.com/tenderly/erigon/erigon-lib/kv"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

func ParseFloat64(str string) float64 {
	v, _ := strconv.ParseFloat(str, 64)
	return v
}

func ChainConfig(tx kv.Tx) *chain.Config {
	genesisBlockHash, err := rawdb.ReadCanonicalHash(tx, 0)
	Check(err)
	chainConfig, err := rawdb.ReadChainConfig(tx, genesisBlockHash)
	Check(err)
	return chainConfig
}
